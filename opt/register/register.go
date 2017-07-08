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
package register

import (
    "fmt"
    "net"
    "regexp"
    "errors"
    "time"
    
    "github.com/duppercloud/trafficrouter/utils"
    "github.com/duppercloud/trafficrouter/ssh"
    "github.com/duppercloud/trafficrouter/portmap"
    log "github.com/Sirupsen/logrus"
)

/*
 *  opt struct
 */
type reg struct {
    opt   string               //option string
    lhost string               //local hostname
    lport string               //local port to connect
    rhost string               //remote host to connect to
    rport string               //remote port to map to
    user string                //remote username
    reader *portmap.Portmap    //portmap handle
    events chan *portmap.Event //portmap event channel
}


var goroutines map[string]chan bool = make(map[string]chan bool, 1)

/*
 *  forEach parser callback
 */
type parsecb func(*reg)

/*
 *  --regiser option parser logic
 */
func parse(str string) (string, string, string) {
    var expr = regexp.MustCompile(`^(.+):([0-9]+|\*)@([^:]+)$`)
	parts := expr.FindStringSubmatch(str)
	if len(str) == 0 {
        utils.Check(errors.New(fmt.Sprintf("Option parse error: [%s]. Format lhost:lport@rhost\n", str)))
	}
        
    return parts[1], parts[2], parts[3]
}

/*
 *  -regiser options iterater
 */
func forEach(opts []string, cb parsecb) {
    for _, opt := range opts {

        r := reg{opt: opt}
        r.lhost, r.lport, r.rhost = parse(opt)

        if r.lport == "*" {
            r.user = r.lhost
        } else {
            r.user = r.lhost + "." + r.lport
        }

        log.Debug("laddr=", r.lhost, ",",
                  "lport=", r.lport, ",",
                  "rhost=", r.rhost, ",",
                  "ruser=", r.user)
        
        // For all port if map file contains entry then connect to mapped ports and wait for new ones.
        r.reader, r.events = portmap.New(r.user, true)            
        log.Debug("mapReader=", r.reader, " event chan =", r.events)

        cb(&r)
    }
}



/*
 *  Connect internal to remote host and periodically check the state.
 */
func connect(r reg, passwd string, interval int, debug bool) {

    // Channel to notify when to stop this go routine
    done := make(chan bool)
    goroutines[r.lport] = done
    
    for {                      
        // Check if host exists
        ipArr, _ := net.LookupHost(r.rhost)
        
        // Connect to all IP address for remote host
        for _, ip := range ipArr {
            hash := r.lhost + "." + r.lport + "@" + ip

            // flag for skipping self connection
            skip := false
            
            // Make sure to not connect to itself for container:* scenario
            laddrs, _ := net.InterfaceAddrs()
            for _, a := range laddrs {
                if ip == a.String() {
                    skip = true
                    break
                }
            }

            // Skip if remote IP is one of local interface ip
            if skip {
                continue 
            }
            
            // connect to dynamic port.
            // store assigned port in map
            // Use the same port for rest of the connections.
            if !ssh.IsConnected(hash) {
                fmt.Println("Connecting...", hash)
                mappedRport := ssh.Connect(r.user, passwd, ip, r.lport, r.rport, hash, debug)
                log.Debug("mappedPort=", mappedRport)
                
                // Add this port mapping to portmap and save it.
                if r.rport == "0" && mappedRport != "0" {
                    r.reader.Add(r.lport, mappedRport)
                    r.rport = mappedRport
                }
            }            
        }

        // Diconnect all ssh connection if channel is closed and return.
        select {
            case _, ok := <- done:
                log.Debug("Terminating goroutine")
                if !ok {
                    for _, ip := range ipArr {
                        hash := r.lhost + "." + r.lport + "@" + ip
                        ssh.Disconnect(hash)
                    }
                    return
                }
            case  <- time.After( time.Duration(interval) * 1000 * time.Millisecond):
            /* no-op */
        }

    }        
}

/*
 *  Connect/Disconnect changes based on portmap
 */
func eventloop(r *reg, passwd string, interval int, debug bool) {
    for {
        event := <-r.events
        if event.Type == portmap.ADDED {
            r.lport = event.Lport
            r.rport = event.Rport
            go connect(*r, passwd, interval, debug)
        } else if event.Type == portmap.DELETED {
            // Disconnect all connections for LPORT by closing goroutine channel.
            close(goroutines[event.Lport])
            delete(goroutines, event.Lport)
        }
    }    
}


/*
 *  Process --regiser options
 */
func Process(passwd string, opts []string, count int, interval int, debug bool) {
    log.Debug(opts)
    
    // Start event loop for each option
    forEach(opts, func(r *reg) {    
        go eventloop(r, passwd, interval, debug)
        
        if r.lport != "*" {
            r.reader.Add(r.lport, "0")
        }
    })      
}