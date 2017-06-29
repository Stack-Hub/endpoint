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
    pmap   map[string]string   //portmap
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
func forEach(opts []string, cbStar parsecb, cbPort parsecb) {
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
        r.reader, r.pmap, r.events = portmap.New(r.user, true)            
        log.Debug("mapReader=", r.reader, " pmap", r.pmap, " event chan =", r.events)

        if r.lport == "*" {
            cbStar(&r)
        } else {
            cbPort(&r)
        }
    }
}

/*
 *  Connect internal to remote host and periodically check the state.
 */
func connect(r *reg, passwd string, interval int, debug bool) {

    // Channel to notify when to stop this go routine
    done := make(chan bool)
    goroutines[r.lport] = done
    
    for {                      
        // Check if host exists
        ipArr, _ := net.LookupHost(r.rhost)
        
        // Diconnect all ssh connection if channel is closed and return.
        select {
            case _, ok := <- done:
                if !ok {
                    for _, ip := range ipArr {
                        hash := r.lhost + "." + r.lport + "@" + ip
                        ssh.Disconnect(hash)
                    }
                    return
                }
            default:
            /* no-op */
        }

        
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

                // Get remote port mapping
                r.rport = r.pmap[r.lport]

                // Use dynamic port mapping if rport is empty
                if r.rport == "" {
                    r.rport = "0"
                }
                log.Debug("rport=", r.rport)

                mappedRport := ssh.Connect(r.user, passwd, ip, r.lport, r.rport, hash, debug)
                log.Debug("mappedPort=", mappedRport)
                
                // Add this port mapping to portmap and save it.
                if r.rport == "0" && mappedRport != "0" {
                    r.reader.Add(r.lport, mappedRport)
                    r.rport = mappedRport
                }
                
            }            
        }

        // sleep between retry intervals
        time.Sleep( time.Duration(interval) * 1000 * time.Millisecond)
    }        
}

/*
 *  Connect/Disconnect changes based on portmap
 */
func procEvents(r *reg, passwd string, interval int, debug bool) {
    for {
        event := <-r.events
        if event.Type == portmap.ADDED {
            r.lport = event.Lport
            r.rport = event.Rport
            go connect(r, passwd, interval, debug)
        } else if event.Type == portmap.DELETED {
            // Disconnect all connections for LPORT by closing goroutine channel.
            close(goroutines[event.Lport])
            delete(goroutines, event.Lport)
        }
    }    
}

/*
 *  Check if this instance of trafficrouter should initiate the connection.
 */
func isAllowed(r *reg) bool {
    // Allow all connections to leader (aka first instance)
    // if not leader then only mapped ports are allowed
    if r.reader.IsLeader() {
        return true
    } else {
        if rport, ok := r.pmap[r.lport]; ok {
            if rport != "0" && rport != ""  {
                return true
            }
        } 
    }
    
    return false
}

/*
 *  Process --regiser options
 */
func Process(passwd string, opts []string, count int, interval int, debug bool) {
    log.Debug(opts)
    
    // For all port, connect to mapped ports then wait for new ports
    connectAll := func(r *reg) {    
        for k, _ := range r.pmap {
            r.lport = k
            if isAllowed(r) {
                go connect(r, passwd, interval, debug)
            }                    
        }
        // Wait for other ports 
        go procEvents(r, passwd, interval, debug)
    }
    
    // For single port, if this instance is first one then connect,
    // else wait for first instance to provide mapped port
    connectOne := func (r *reg) {
        if isAllowed(r) {
            go connect(r, passwd, interval, debug)
        } else {
            // Wait for leader to provide mapped port 
            go procEvents(r, passwd, interval, debug)
        }        
    } 
    
    forEach(opts, connectAll, connectOne)      
}