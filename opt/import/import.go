package Import

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"

	"github.com/prometheus/common/log"
	"github.com/microstacks/stack/endpoint/omap"
	"github.com/microstacks/stack/endpoint/server"
	"github.com/microstacks/stack/endpoint/utils"
)

/*
 *  opt struct
 */
type Import struct {
	opt   string        //option string
	lhost string        //local hostname
	lport string        //local port to connect
	rhost string        //remote host that connects
	rport string        //remote port to map to
	user  string        //username
	block bool          //Block process till service connects
	lb    *net.Listener //Listener socket for load balancer
}

type parsecb func(*Import)
type callback func()

var serverRegistered bool = false
var asyncCB callback = nil
var imports []Import
var done bool = false

func Cleanup() {
}

/*
 * forEach parser callback
 */
func forEach(opts []string, cb parsecb) {

	// Prepare and store all require option in global context.
	for _, opt := range opts {
		var i Import
		i.block, i.rhost, i.rport, i.lhost, i.lport = parse(opt)

		i.user = i.rhost
		log.Debug("i.user=", i.user)

		if i.rport != "*" {
			i.user += "." + i.rport
		}

		log.Debug("block=", i.block, ",",
			"raddr=", i.rhost, ",",
			"rport=", i.rport, ",",
			"laddr=", i.lhost, ",",
			"lport=", i.lport)
		imports = append(imports, i)
	}

	// Trigger callback for each require option.
	for idx, _ := range imports {
		cb(&imports[idx])
	}
}

/*
 * parse --import option
 * Formats rhost:rport             - one2one port mapping
 *         rhost:rport@lhost:lport - load balance rport to lport
 */
func parse(str string) (bool, string, string, string, string) {
	var expr = regexp.MustCompile(`^(\^)?([^:]+):([0-9]+|\*)(@([a-zA-Z][a-zA-Z0-9]+|\*):([0-9]+))?$`)
	parts := expr.FindStringSubmatch(str)

	if len(parts) == 0 {
		utils.Check(errors.New(fmt.Sprintf("Require option parse error: [%s]. Format rhost:rport[@lhost:lport]\n", str)))
	}

	block := false
	if parts[1] == "^" {
		block = true
	}
	return block, parts[2], parts[3], parts[5], parts[6]
}

/*
 * Connection added evant callback
 * When all services connect, invoke async callback
 */
func ConnAddEv(m *omap.OMap, h *utils.Host) {

	// If this is first connection then decrement callback ticker
	i := m.Userdata.(*Import)
	i.block = false

	// TODO: Change key such that it is unique/connections.
	// Currently it is based on random port assignment, which can overlap with
	// different localhost/8 IP
	m.Add(h.LocalPort, h)

	payload, err := json.Marshal(h)
	utils.Check(err)
	fmt.Println("Connected", string(payload))

	// trigger on-connect code block
	utils.OnConnect(h.RemoteIP,
		fmt.Sprint(h.RemotePort),
		h.LocalIP,
		fmt.Sprint(h.LocalPort))

	// If this is first connection start listening on load balanced port
	if len(i.lhost) > 0 && i.lb == nil {
		i.lb = listen(m, i.lhost, i.lport)
	}

	log.Debug("done=", done)
	// Invoke callback after all required services are connected.
	if !done {
		// Are all service connected?
		for _, i := range imports {
			log.Debug("checking r=", i)
			if i.block {
				return
			}
		}

		if asyncCB != nil {
			done = true
			log.Debug("Invoking CB", asyncCB)
			asyncCB()
			asyncCB = nil
		}
	}
}

/*
 * Connection removed callback
 */
func ConnRemoveEv(m *omap.OMap, h *utils.Host) {
	m.Remove(h.LocalPort)

	payload, err := json.Marshal(h)
	utils.Check(err)
	fmt.Println("Disconnected", string(payload))

	// trigger on-connect code block
	utils.OnDisconnect(h.RemoteIP,
		fmt.Sprint(h.RemotePort),
		h.LocalIP,
		fmt.Sprint(h.LocalPort))

}

func listen(m *omap.OMap, lhost string, lport string) *net.Listener {

	ipAddr := utils.GetIP(lhost)

	addr := fmt.Sprintf("%s:%s", ipAddr.String(), lport)
	log.Debug("addr=", addr)

	fmt.Printf("Listening on %s\n", addr)
	l, err := net.Listen("tcp", addr)
	utils.Check(err)

	go func() {
		for {
			// Listen for an incoming connection.
			conn, err := l.Accept()
			utils.Check(err)

			// Handle connections in a new goroutine.
			go handleRequest(m, conn)
		}
	}()

	return &l
}

/*
 * Data Handling
 * Handles incoming tcp requests and
 * route to next available connection.
 */
func handleRequest(m *omap.OMap, in net.Conn) {
	defer in.Close()

	for {
		el := m.Next()
		if el != nil {
			h := el.Value.(*utils.Host)
			if h != nil {

				endpoint := utils.Endpoint{
					Host: h.LocalIP,
					Port: h.LocalPort,
				}

				log.Debug("Connecting to", endpoint.String())
				out, err := net.Dial("tcp", endpoint.String())
				// Connection failed, remove connection information from the list
				if err != nil {
					log.Error(err)
					log.Debug("Connection failed removing ", el)
					continue
				}
				defer out.Close()

				log.Debug("Routing Data for ", h)
				go io.Copy(out, in)
				io.Copy(in, out)
			}
		}
		break
	}

}

/*
 * Process require options
 */
func Process(passwd string, opts []string, cb callback) {
	log.Debug(opts)

	if !serverRegistered {
		// Start SSH Server
		go server.Listen()
		serverRegistered = true
	}

	forEach(opts, func(i *Import) {
		// Initialize Ordered map and server events.
		m := omap.New()
		m.Userdata = i

		// Add user to ssh server
		go server.AddUser(i.user, m, ConnAddEv, ConnRemoveEv)

	})

	// Check if callback can be invoked or need to wait for specific services to connect.
	for _, i := range imports {
		if i.block {
			asyncCB = cb
			cb = nil
			break
		}
	}

	// Invoke callback
	if cb != nil {
		cb()
	}

}
