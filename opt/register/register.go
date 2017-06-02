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

var goroutines map[string]chan bool = make(map[string]chan bool, 1)

/*
 *  forEach parser callback
 */
type parsecb func(string, string, string, string)

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
        lhost, lport, rhost := parse(opt)

        log.Debug("laddr=", lhost, ",",
                  "lport=", lport, ",",
                  "rhost=", rhost)
        
        cb(opt, lhost, lport, rhost)
    }
}

/*
 *  make goroutine channel and Connect
 */
func connect(opt string, lhost string, lport string, rhost string, rport string,
             passwd string, interval int, mapReader *portmap.Portmap, debug bool) {

    done := make(chan bool)
    goroutines[lport] = done
    iconnect(opt, lhost, lport, rhost, rport, passwd, interval, mapReader, debug, done)
} 

/*
 *  Connect internal to remote host and periodically check the state.
 */
func iconnect(opt string, lhost string, lport string, rhost string, rport string,
             passwd string, interval int, mapReader *portmap.Portmap, debug bool, isClosed chan bool) {

    uname := lhost + "." + lport
    for {                  
        
        //Check if host exists
        ipArr, _ := net.LookupHost(rhost)
        
        // Diconnect all ssh connection if channel is closed and return.
        select {
            case _, ok := <- isClosed:
                if !ok {
                    for _, ip := range ipArr {
                        addr := uname + "@" + ip
                        ssh.Disconnect(addr)
                    }
                    return
                }
            default:
            /* no-op */
        }

        
        for _, ip := range ipArr {
            addr := uname + "@" + ip

            skip := false
            
            //Make sure to not connect to itself for nodes:* scenario
            laddrs, _ := net.InterfaceAddrs()
            for _, a := range laddrs {
                if ip == a.String() {
                    skip = true
                    break
                }
            }

            // Skip is remote IP is one of local interface ip
            if skip {
                continue 
            }
            
            // connect to dynamic port.
            // store assigned port in map
            // Use the same port for rest of the connections.
            if !ssh.IsConnected(addr) {
                fmt.Println("Connecting...", addr)
                mappedRport := ssh.Connect(uname, passwd, ip, lport, rport, debug)
                
                // Add this port mapping to portmap and save it.
                if rport == "0" {
                    mapReader.Add(lport, mappedRport)
                    rport = mappedRport
                }
                
            }            
        }

        time.Sleep( time.Duration(interval) * 1000 * time.Millisecond)
    }        
}

/*
 *  Check if this instance of trafficrouter should initiate the connection.
 */
func isAllowed(reader *portmap.Portmap, pmap map[string]string, lport string) bool {

    if rport, ok := pmap[lport]; ok {
        if rport == "0" {
            if reader.IsLeader() {
                return true
            }            
        } else {
            return true
        }
    }
    return false
}

/*
 *  Process --regiser options
 */
func procEvents(opt string, lhost string, rhost string,
                passwd string, interval int, debug bool, mapReader *portmap.Portmap, events chan *portmap.Event) {
    for {
        event := <-events
        if event.Type == portmap.ADDED {
            go connect(opt, lhost, event.Lport, rhost, event.Rport, passwd, interval, mapReader, debug)
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
    
    forEach(opts, func(opt string, lhost string, lport string, rhost string) {        
        // For all port if map file contains entry then connect to mapped ports and wait for new ones.
        mapReader, pmap, events := portmap.New(lhost + "." + lport)            

        if lport == "*" {
            for k, v := range pmap {
                if isAllowed(mapReader, pmap, k) {
                    go connect(opt, lhost, lport, rhost, v, passwd, interval, mapReader, debug)
                }                    
            }
            
            // Wait for other ports 
            go procEvents(opt, lhost, rhost, passwd, interval, debug, mapReader, events)
            
        } else {
            if isAllowed(mapReader, pmap, lport) {
                rport := pmap[lport]
                go connect(opt, lhost, lport, rhost, rport, passwd, interval, mapReader, debug)
            } else {
                // Wait for other ports 
                go procEvents(opt, lhost, rhost, passwd, interval, debug, mapReader, events)
            }
        }
    })       
}