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
    "os"
    "io"
    "strconv"
    "regexp"
    "errors"
    "syscall"
    "strings"
    "os/exec"
    
    "./omap"
    "./user"
    "./server"
    "./utils"
    log "github.com/Sirupsen/logrus"
    "github.com/urfave/cli"
)

type callback func(string, string, string, string)

func split(opts string){
    return strings.Split(opt, ",")
}

func join(opts []string) string{
    return strings.Join(opts, ",")
}

func forEach(opts string, cb callback) {
    optArr := split(opts)

    for _, opt := range optArr {
        rhost, rport, lhost, lport := parse(opt)

        // User remote port for listening if no local port provided
        if lport == "" {
            lport = rport
        }

        log.Debug("raddr=", rhost, ",",
                  "rport=", rport, ",",
                  "laddr=", lhost, ",",
                  "lport=", lport)
        
        cb(rhost, rport, lhost, lport)
    }
}

func parse(str string) (string, string, string, string) {
    var expr = regexp.MustCompile(`^(.+):([0-9]+)@([^:]+)(:([0-9]+))?$`)
	parts := expr.FindStringSubmatch(str)
	if len(str) == 0 {
        utils.Check(errors.New(fmt.Sprintf("Option parse error: [%s]. Format rhost:rport@lhost(:lport)?\n", str)))
	}
    
    return parts[1], parts[2], parts[3], parts[5]
}


// Handles incoming requests.
func handleRequest(m *omap.OMap, in net.Conn) {
    defer in.Close()

    for {
        h := m.Next()
        if h == nil {
            // Send a response back to person contacting us.
            in.Close()
        } else {
            port := strconv.Itoa(int(h.Value.(utils.Host).ListenPort))
            out, err := net.Dial("tcp", "127.0.0.1:" + port)

            // Connection failed, remove connection information from the list
            if err != nil {
                m.Remove(h)
                continue
            }

            go io.Copy(out, in)
            io.Copy(in, out)
            defer out.Close()    
        }
        break
    } 
}

func ConnAddEv(m *omap.OMap, p int, h *utils.Host) {
    m.Add(p, h)
    fmt.Printf("Connected %s:%d on Port %d\n", 
               h.RemoteIP, 
               h.AppPort, 
               h.ListenPort)
}

func ConnRemoveEv(m *omap.OMap, p int, h *utils.Host) {
    m.Remove(p)
    fmt.Printf("Removed %s:%d from Port %d\n", 
               h.RemoteIP, 
               h.AppPort, 
               h.ListenPort)
}

func process(c *cli.Context, opts string) {

    // Get user options
    swamped := c.String("swmp")
    log.Debug("swamped=", swamped)

    //Map to store Usernames
    unames := make(map [string]string, 1)
    
    if swamped {
        // Get password
        passwd := c.String("passwd")

        forEach(opt, func(rhost string, rport string, lhost string, lport string) {
            uname := rhost + "." + rport
            log.Debug("uname=", uname)

            //Create User
            u := user.NewUserWithPassword(uname, passwd)
            log.Debug("user=", u)
            defer u.Delete()

            // Initialize Ordered map and server events.
            m := omap.New()
            go server.Monitor(ConnAddEv, ConnRemoveEv)

            addr := fmt.Sprintf("%s:%s", lhost, lport)
            log.Debug("addr=", addr)

            fmt.Printf("Listening on %s\n", addr)
            l, err := net.Listen("tcp", addr)
            utils.Check(err)

            // Close the listener when the application closes.
            defer l.Close()

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
        
    } else {
        
        forEach(opt, func(rhost string, rport string, lhost string, lport string) {
            uname := rhost + "." + rport
            log.Debug("uname=", uname)

            // Form username based on rhost.rport and add to map
            unames[opt] := uname
        })        

        // Swamp new exec with user argument. 
        // There arguments are used for local service discovery by clients
        args := make([]string, len(os.Args) + 2)
        copy(args, os.Args)
        args[len(os.Args)]     = "-u"
        args[len(os.Args) + 1] = join(uname)

        log.Println("Swamping new exec")
        execf, err := osext.Executable()
        utils.Check(err)
        err = syscall.Exec(execf, args, os.Environ())

        //This shouln't be executed
        utils.Check(err)                    
    }
    

}