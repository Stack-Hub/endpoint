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
package main

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
    "os/signal"
    "time"
    "bufio"
    
    "./omap"
    "./client"
    "./forcecmd"
    "./user"
    "./server"
    "./utils"
    log "github.com/Sirupsen/logrus"
    "github.com/urfave/cli"
    "github.com/kardianos/osext"
)

var m * omap.OMap
var ch chan bool
var registered bool 


/*
 *  Cleanup before exit
 */
func cleanup() {    
    log.Debug("Cleaning up")
    forcecmd.Cleanup()
    user.Cleanup()
}

/*
 *  Install Signal handler for proper cleanup.
 */
func installHandler() {
    sigs := make(chan os.Signal, 1)    
    signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        sig := <-sigs
        log.Debug(sig)
        cleanup()
        os.Exit(1)
    }()    
}


// Handles incoming requests.
func handleRequest(in net.Conn) {
    defer in.Close()

    h := m.Next()
    if h == nil {
        // Send a response back to person contacting us.
        in.Write([]byte("No Routes available."))   
    } else {
        port := strconv.Itoa(int(h.Value.(utils.Host).ListenPort))
        out, _ := net.Dial("tcp", "127.0.0.1:" + port)
        go io.Copy(out, in)
        io.Copy(in, out)
        defer out.Close()    
    }
}

func ConnAddEv(p int, h *utils.Host) {
    m.Add(p, h)
    fmt.Printf("Connected %s:%d on Port %d\n", 
               h.RemoteIP, 
               h.AppPort, 
               h.ListenPort)
            
    if registered == false {
        // Unblock registeration
        log.Debug("Unblocking Registration")
        ch <-true
    }
    
    
}

func ConnRemoveEv(p int, h *utils.Host) {
    m.Remove(p)
    fmt.Printf("Removed %s:%d from Port %d\n", 
               h.RemoteIP, 
               h.AppPort, 
               h.ListenPort)
}

func parseNeeds(str string) (string, string, string, string) {
    var expr = regexp.MustCompile(`^(.+):([0-9]+)@([^:]+)(:([0-9]+))?$`)
	parts := expr.FindStringSubmatch(str)
	if len(str) == 0 {
        utils.Check(errors.New(fmt.Sprintf("Option parse error: [%s]. Format rhost:rport@lhost(:lport)?\n", str)))
	}
    
    if len(parts) > 4 {
        return parts[1], parts[2], parts[3], parts[5]
    } else {
        return parts[1], parts[2], parts[3], ""
    }
}

func parseRegister(str string) (string, string, string, bool) {
    var expr = regexp.MustCompile(`^(.+):([0-9]+)@([^:\*]+)(\*)?$`)
	parts := expr.FindStringSubmatch(str)
	if len(str) == 0 {
        utils.Check(errors.New(fmt.Sprintf("Option parse error: [%s]. Format lhost:lport@rhost(*)?\n", str)))
	}
    
    wildcard := false
    
    log.Debug(parts)
    if parts[4] == "*" {
        wildcard = true
    }
    
    return parts[1], parts[2], parts[3], wildcard    
}


func TrafficRouter(c *cli.Context) error {

    /*
     * Install Signal handlers for proper cleanup.
     */
    installHandler()
    
    /*
     * Global error handling.
     * Cleanup and exit.
     */
    defer func() {
        if r := recover(); r != nil {
            log.Debug("Recovered ", r)
            cleanup()
            os.Exit(1)
        }
    }()    
    
    ch = make(chan bool, 2)
    registered = false
    
    // Send message for Force Command mode and return.
    if (c.Bool("f") == true) {
        forcecmd.SendConfig()
        return nil
    }

    passwd := c.String("passwd")
    if passwd == "" {
        log.Fatal("Empty password. Please provide password")
    }
        
    // Wait for Needed service before registering.
    if (c.String("needs") != "") {
        rhost, rport, lhost, lport := parseNeeds(c.String("needs"))

        // User remote port for listening if no local port provided
        if lport == "" {
            lport = rport
        }
        log.Debug("raddr=", rhost, ",rport=", rport, ",laddr=", lhost, ",lport=", lport)

        // Get user options
        usr  := c.String("usr")
        uid  := c.Int("uid")
        mode := c.Int("mode")
        log.Debug("usr=", usr, ",uid=", uid, ",mode=", mode)

        // No user options indicate user invocation.
        // Create user and invoke again with user information.
        if usr == "" {
            // Create new user
            uname  := rhost + "." + rport
            u := user.NewUserWithPassword(uname, passwd)
            log.Debug("user=", u)

            args := make([]string, len(os.Args) + 6)
            copy(args, os.Args)
            args[len(os.Args)]     = "-u"
            args[len(os.Args) + 1] = u.Name
            args[len(os.Args) + 2] = "-uid"
            args[len(os.Args) + 3] = strconv.Itoa(u.Uid)
            args[len(os.Args) + 4] = "-mode"
            args[len(os.Args) + 5] = strconv.Itoa(u.Mode)

            // Swamp new exec with user argument. 
            // There arguments are used for local service discovery by clients
            log.Println("Swamping new exec")
            execf, err := osext.Executable()
            utils.Check(err)
            err = syscall.Exec(execf, args, os.Environ())
            utils.Check(err)            
        }
        
        u := &user.User{Name: usr, Uid: uid, Mode: mode}
        user.RestoreUser(u)
        defer u.Delete()
        
        // Initialize Ordered map and server events.
        m = omap.New()
        go server.Monitor(ConnAddEv, ConnRemoveEv)
        
        addr := fmt.Sprintf("%s:%s", lhost, lport)
        log.Debug("addr=", addr)
        
        l, err := net.Listen("tcp", addr)
        utils.Check(err)
        
        // Close the listener when the application closes.
        defer l.Close()

        go func() {
            fmt.Printf("Listening on %s\n", addr)
            for {
                // Listen for an incoming connection.
                conn, err := l.Accept()
                utils.Check(err)

                // Handle connections in a new goroutine.
                go handleRequest(conn)
            }                
        }()
    } else {
        // Unblock registeration
        log.Debug("Unblocking Registration")
        ch <-true
    }
    
    go func() {
        /* Client mode */
        if (c.String("register") != "") {
            log.Debug("Waiting for Dependency connections")
            <-ch 
            log.Debug("Registering")
            lhost, lport, rhost, wc := parseRegister(c.String("register"))

            log.Println(lhost, lport, rhost, wc)
            uname := lhost + "." + lport
            
            if wc == true {
                i := 1
                for {
                    rdns := rhost + strconv.Itoa(i)
                    //Check if host exists
                    log.Debug("Checking ", rdns)
                    raddrs, err := net.LookupHost(rdns)
                    if err != nil && len(raddrs) > 0 {
                        cmd := client.StartWithPasswd(uname, passwd, rdns, lport)
                        defer client.Stop(cmd)
                        log.Debug(cmd)          
                        i++
                    } 
                    poll := c.Int("poll-interval")
                    time.Sleep( time.Duration(poll) * 1000 * time.Millisecond)
                }
            } else {
                cmd := client.StartWithPasswd(uname, passwd, rhost, lport)
                defer client.Stop(cmd)
                log.Debug(cmd)                                                    
            }
            
            utils.BlockForever()
        }         
    }()
    
    if (c.String("cmd") != "") {
        cmdargs := strings.Split(c.String("cmd")," ")
        cmd := cmdargs[0]
        args := cmdargs[1:]
        log.Debug("Executing ", cmd, args)
        c := exec.Command(cmd, args...)
        stdout, _ := c.StdoutPipe()
        stderr, _ := c.StderrPipe()
        
        stdoutscanner := bufio.NewScanner(stdout)
        stderrscanner := bufio.NewScanner(stderr)
        go func() {
            for stdoutscanner.Scan() {
                fmt.Println(stdoutscanner.Text())
            }
        }()
        go func() {
            for stderrscanner.Scan() {
                fmt.Println(stderrscanner.Text())
            }
        }()

        
        c.Start()
    }
    
    utils.BlockForever()
    return nil
}