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
    "os"
    "strconv"
    "regexp"
    "errors"
    "encoding/json"
    
    "github.com/duppercloud/trafficrouter/omap"
    "github.com/duppercloud/trafficrouter/server"
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
    count int
    rhost string               //remote host that connects
    rport string               //remote port to map to
    user  string               //username
    lb    *net.Listener        //Listener socket for load balancer
}

type parsecb func(*req)
type callback func()

var done bool = false
var asyncCB callback = nil 
var reqs []req

func Cleanup() {
    done = false
}

/*
 * forEach parser callback
 */
func forEach(opts []string, cb parsecb) {

    // Prepare and store all require option in global context.
    for _, opt := range opts {
        var r req
        r.rhost, r.rport, r.count, r.lhost, r.lport = parse(opt)

        r.user = r.rhost
        log.Debug("r.user=", r.user)

        if r.rport != "*" {
            r.user += "." + r.rport
        }
        
        log.Debug("raddr=", r.rhost, ",",
                  "rport=", r.rport, ",",
                  "count=", r.count, ",",
                  "laddr=", r.lhost, ",",
                  "lport=", r.lport)
        reqs = append(reqs, r)
    }
    
    // Trigger callback for each require option.
    for idx, _ := range reqs {
        cb(&reqs[idx])
    }
}

/*
 * parse --require option
 * Formats rhost:rport             - one2one port mapping
 *         rhost:rport@lhost:lport - load balance rport to lport
 */
func parse(str string) (string, string, int, string, string) {
    var expr = regexp.MustCompile(`^([^:]+):([0-9]+|\*)(@([a-zA-Z][a-zA-Z0-9]+|\*):([0-9]+))?$`)
	parts := expr.FindStringSubmatch(str)
    
	if len(parts) == 0 {
        utils.Check(errors.New(fmt.Sprintf("Require option parse error: [%s]. Format rhost:rport[@lhost:lport]\n", str)))
	}
    
    count := 1
    
    if parts[2] == "*" {
        count = 0    
    } 
        
    return parts[1], parts[2], count, parts[4], parts[5]
}

/*
 * Connection added evant callback
 * When all services connect, invoke async callback
 */
func ConnAddEv(m *omap.OMap, h *utils.Host) {

    // If this is first connection then decrement callback ticker
    r := m.Userdata.(*req)
    if r.count > 0 {
        r.count--
        log.Debug("r=", r)
    }
    
    m.Add(h.Config.Instance, h)
    
    payload, err := json.Marshal(h)
    utils.Check(err)
    fmt.Println("Connected", string(payload))

    // trigger on-connect code block
    utils.OnConnect(h.RemoteIP, 
                    fmt.Sprint(h.Config.Port), 
                    os.Getenv("BINDADDR"), 
                    fmt.Sprint(h.ListenPort), 
                    fmt.Sprint(h.Config.Instance),
                    fmt.Sprint(h.Config.Label))
    
    // If this is first connection start listening on load balanced port
    if len(r.lhost) > 0 && m.Len() == 1 && r.lb == nil {
        r.lb = listen(m, r.lhost, r.lport)
    }
    
    log.Debug("done=", done)
    // If asyncCB is not triggered check if all requirements are met.
    if !done {
        // Return if not all required connections are established
        for _, r := range reqs {
            log.Debug("checking r=", r)
            if r.count > 0 {
                return
            }
        }

        // Trigger callback because all required connections are established.
        if asyncCB != nil {
            done = true
            asyncCB()
        }        
    }
}

/*
 * Connection removed callback
 */
func ConnRemoveEv(m *omap.OMap, h *utils.Host) {
    m.Remove(h.Config.Instance)

    payload, err := json.Marshal(h)
    utils.Check(err)
    fmt.Println("Disconnected", string(payload))

    // trigger on-connect code block
    utils.OnDisconnect(h.RemoteIP, 
                       fmt.Sprint(h.Config.Port), 
                       "127.0.0.1", 
                       fmt.Sprint(h.ListenPort), 
                       fmt.Sprint(h.Config.Instance), 
                       fmt.Sprint(h.Config.Label))

}

func listen(m *omap.OMap, lhost string, lport string) (*net.Listener) {

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
                port := strconv.Itoa(int(h.ListenPort))

                log.Debug("Connecting to localhost:", port)
                out, err := net.Dial("tcp", "127.0.0.1:" + port)
                // Connection failed, remove connection information from the list
                if err != nil {
                    log.Error(err)
                    log.Debug("Connection failed removing ", el)
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
    
    // Start SSH Server
    go server.Listen()
    
    forEach(opts, func(r *req) {
        // Initialize Ordered map and server events.
        m := omap.New()   
        m.Userdata = r
                
        // Add user to ssh server
        go server.AddUser(r.user, m, ConnAddEv, ConnRemoveEv)

        //Store callback for later invocation
        if cb != nil {
            asyncCB = cb
            cb = nil            
        } 
    })
    
    // If options are non-zero then don't invoke callback now.
    // AsyncCB will be invoked when all the services connects.
    if cb != nil {
        cb()
    }
}