/* Copyright 2017, Ashish Thakwani. 
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.LICENSE file.
 */
package require

import (
    "fmt"
    "net"
    "io"
    "strconv"
    "regexp"
    "errors"
    
    "../../omap"
    "../../user"
    "../../server"
    "../../utils"
    log "github.com/Sirupsen/logrus"
    "github.com/urfave/cli"
)

type parsecb func(string, string, string, string, string)
type callback func()

var cbTicker int = 0
var asyncCB callback = nil 

func forEach(opts []string, cb parsecb) {

    for _, opt := range opts {
        rhost, rport, lhost, lport := parse(opt)

        // User remote port for listening if no local port provided
        if lport == "" {
            lport = rport
        }

        log.Debug("raddr=", rhost, ",",
                  "rport=", rport, ",",
                  "laddr=", lhost, ",",
                  "lport=", lport)
        
        cb(opt, rhost, rport, lhost, lport)
    }
}

func parse(str string) (string, string, string, string) {
    var expr = regexp.MustCompile(`^(.+):([0-9]+)@([^:]+)(:([0-9]+))?$`)
	parts := expr.FindStringSubmatch(str)
	if len(str) == 0 {
        utils.Check(errors.New(fmt.Sprintf("Require option parse error: [%s]. Format rhost:rport@lhost(:lport)?\n", str)))
	}
    
    return parts[1], parts[2], parts[3], parts[5]
}


// Handles incoming requests.
func handleRequest(m *omap.OMap, in net.Conn) {
    defer in.Close()

    for {
        h := m.Next()
        if h != nil {
            port := strconv.Itoa(int(h.Value.(*utils.Host).ListenPort))
            out, err := net.Dial("tcp", "127.0.0.1:" + port)
            defer out.Close()    

            // Connection failed, remove connection information from the list
            if err != nil {
                m.RemoveEl(h)
                continue
            }

            go io.Copy(out, in)
            io.Copy(in, out)
        }
        break
    }
    
}

func ConnAddEv(m *omap.OMap, uname string, p int, h *utils.Host) {

    // If this is first connection then decrement callback ticker
    if m.Len() == 0 {
        cbTicker--
    }
    
    m.Add(p, h)
    fmt.Printf("Connected %s from %s:%d at Port %d\n", 
               uname, 
               h.RemoteIP, 
               h.AppPort, 
               h.ListenPort)
    
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

func ConnRemoveEv(m *omap.OMap, uname string, p int, h *utils.Host) {
    m.Remove(p)
    fmt.Printf("Removed %s from %s:%d at Port %d\n", 
               uname, 
               h.RemoteIP, 
               h.AppPort, 
               h.ListenPort)
}

func Process(c *cli.Context, cb callback) {

    opts := c.StringSlice("require")
    log.Debug(opts)
    
    // Get password
    passwd := c.String("passwd")
        
    forEach(opts, func(opt string, rhost string, rport string, lhost string, lport string) {

        //Store callback for later invocation
        if cb != nil {
            asyncCB = cb
            cb = nil            
        }

        uname := rhost + "." + rport
        log.Debug("uname=", uname)

        //Create User
        u := user.NewUserWithPassword(uname, passwd)
        log.Debug("user=", u)

        // Initialize Ordered map and server events.
        m := omap.New()

        // Increment callback ticker.
        cbTicker++

        // Monitor unix listening socker based on uname
        go server.Monitor(m, uname, ConnAddEv, ConnRemoveEv)

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
    })

    // If options are non-zero then don't invoke callback now.
    // AsyncCB will be invoked when all the services connects.
    if cb != nil {
        cb()
    }
    

}