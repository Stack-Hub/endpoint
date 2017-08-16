/* Copyright (C) Ashish Thakwani - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Ashish Thakwani <athakwani@gmail.com>, August 2017
 */
package require

import (
    "fmt"
    "net"
    "io"
    "strconv"
    "regexp"
    "errors"
    "encoding/json"
    
    "github.com/duppercloud/trafficrouter/omap"
    "github.com/duppercloud/trafficrouter/user"
    "github.com/duppercloud/trafficrouter/monitor"
    "github.com/duppercloud/trafficrouter/utils"
    log "github.com/Sirupsen/logrus"
)

/*
 *  opt struct
 */
type req struct {
    opt   string               //option string
    lhost string               //local hostname
    lport string               //local port to connect
    rhost string               //remote host to connect to
    rport string               //remote port to map to
    user  string               //username
    lb    *net.Listener        //Listener socket for load balancer
}

type parsecb func(req)
type callback func()

var cbTicker int = 0
var asyncCB callback = nil 

/*
 * forEach parser callback
 */
func forEach(opts []string, cb parsecb) {

    for _, opt := range opts {
        var r req
        r.rhost, r.rport, r.lhost, r.lport = parse(opt)

        r.user = r.rhost
        log.Debug("r.user=", r.user)

        if r.rport != "*" {
            r.user += "." + r.rport
        }
        
        log.Debug("raddr=", r.rhost, ",",
                  "rport=", r.rport, ",",
                  "laddr=", r.lhost, ",",
                  "lport=", r.lport)
        
        cb(r)
    }
}

/*
 * parse --require option
 * Formats rhost:rport             - one2one port mapping
 *         rhost:rport>lhost:lport - load balance rport to lport
 */
func parse(str string) (string, string, string, string) {
    var expr = regexp.MustCompile(`^([^:]+):([0-9]+|\*)(>((?:(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])\.){3}(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])):([0-9]+))?$`)
	parts := expr.FindStringSubmatch(str)
    
	if len(str) == 0 {
        utils.Check(errors.New(fmt.Sprintf("Require option parse error: [%s]. Format rhost:rport[>lhost:lport]\n", str)))
	}
        
    return parts[1], parts[2], parts[4], parts[5]
}


/*
* Get interface IP address
 */
func getIP(iface string) (*net.TCPAddr, error) {
    
    ief, err := net.InterfaceByName(iface)
    if err !=nil{
        return nil, err
    }

    addrs, err := ief.Addrs()
    if err !=nil{
        return nil, err
    }

    tcpAddr := &net.TCPAddr{
        IP: addrs[0].(*net.IPNet).IP,
    }    
    
    return tcpAddr, nil
}

/*
 * Connection added evant callback
 * When all services connect, invoke async callback
 */
func ConnAddEv(m *omap.OMap, uname string, p int, h *utils.Host) {

    // If this is first connection then decrement callback ticker
    if m.Len() == 0 {
        cbTicker--
    }
    
    m.Add(p, h)
    
    payload, err := json.Marshal(h)
    utils.Check(err)
    fmt.Println("Connected", string(payload))
    
    // If this is first connection start listening on load balanced port
    r := m.Userdata.(req)
    if len(r.lhost) > 0 && m.Len() == 1 && r.lb == nil {
        r.lb = listen(m, r.lhost, r.lport)
        m.Userdata = r
    }
    
    // All required connections are established
    // inform the caller with async callback and 
    // unset cbTicker.
    if cbTicker == 0 {
        cbTicker = -1
        log.Debug("cbTicker =", cbTicker)
        log.Debug(asyncCB)
        if asyncCB != nil {
            asyncCB()
        }
    }
    
}

/*
 * Connection removed callback
 */
func ConnRemoveEv(m *omap.OMap, uname string, p int, h *utils.Host) {
    m.Remove(p)

    payload, err := json.Marshal(h)
    utils.Check(err)
    fmt.Println("Removed", string(payload))
}

func listen(m *omap.OMap, lhost string, lport string) (*net.Listener) {
    addr := fmt.Sprintf("%s:%s", lhost, lport)
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
                port := strconv.Itoa(int(h.ListenPort))

                /*
                 * Bind to eth0 IP address.
                 * This is crucial for self discovery.
                 * Some servers connects back to client at specific ports.
                 * This allows them to directly reach the client at it's IP address.
                 */

                tcpAddr, err := getIP("eth0")
                if err != nil {
                    tcpAddr = &net.TCPAddr{
                        IP: []byte{127,0,0,1},
                    }    
                }
                
                log.Debug("Binding to ", tcpAddr)
                
                d := net.Dialer{LocalAddr: tcpAddr}
                out, err := d.Dial("tcp", "127.0.0.1:" + port)
                // Connection failed, remove connection information from the list
                if err != nil {
                    m.RemoveEl(el)
                    continue
                }
                defer out.Close()    

                log.Debug("Routing Data for ", h.Uname, " to host ", h)
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
func Process(passwd string, opts []string,  cb callback) {
    log.Debug(opts)
            
    forEach(opts, func(r req) {

        //Store callback for later invocation
        if cb != nil {
            asyncCB = cb
            cb = nil            
        }

        //Create User
        u := user.New(r.user, passwd)
        log.Debug("user=", u)

        // Initialize Ordered map and server events.
        m := omap.New()   
        m.Userdata = r

        // Increment callback ticker.
        cbTicker++

        // Monitor unix listening socker based on uname
        go server.Monitor(m, r.user, ConnAddEv, ConnRemoveEv)

    })

    // If options are non-zero then don't invoke callback now.
    // AsyncCB will be invoked when all the services connects.
    if cb != nil {
        cb()
    }
}