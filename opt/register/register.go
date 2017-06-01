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
    
    part7 := ""
    if parts[5] != "?" {
        part7 = parts[7]
    }
    
    return parts[1], parts[2], parts[3], parts[5][0], part7
}

/*
 *  -regiser options iterater
 */
func forEach(opts []string, cb parsecb) {
    for _, opt := range opts {
        lhost, lport, rhost := parse(opt)

        log.Debug("laddr=", lhost, ",",
                  "lport=", lport, ",",
                  "rport=", rport)
        
        cb(opt, lhost, lport, rhost)
    }
}

/*
 *  Connect to remote host and periodically check the state.
 */
func connect(opt string, lhost string, lport string, rhost string,
             passwd string, interval int, debug bool) {

    uname := lhost + "." + lport
    for {                  

        //if the port map is not present in map file {
        //  return
        //}
        
        //Check if host exists
        ipArr, _ := net.LookupHost(rhost)
        
        for _, ip := range ipArr {
            addr := uname + "@" + ip

            //Make sure to not connect to itself for nodes:* scenario

            // connect to dynamic port.
            // store assigned port in map
            // Use the same port for rest of the connections.
            
            if !ssh.IsConnected(addr) {
                fmt.Println("Connecting...", addr)
                ssh.Connect(uname, passwd, ip, lport, debug)
            }            
        }

        time.Sleep( time.Duration(interval) * 1000 * time.Millisecond)
    }        
}

func isAllowed(reader *portmap.Portmap, pmap map[string]int, lport string) bool {

    if rport, ok := pmap[lport]; ok {
        return true
    } else {
        if reader.IsLeader() {
            return true
        }
    }
    return false
}

func procEvents(opt string, lhost string, rhost string,
                passwd string, interval int, debug bool, events chan *portmap.Event) {
    for {
        event := <-events
        if event.Type == portmap.ADDED {
            go connect(opt, lhost, event.Lport, rhost, passwd, interval, debug)
        } else if event.Type == portmap.DELETED {
            // Disconnect 
        }
    }    
}

/*
 *  Process --regiser options
 */
func Process(passwd string, opts []string, count int, interval int, debug bool) {
    log.Debug(opts)
    
    forEach(opts, func(opt string, lhost string, lport string, rhost string) {
        mapReader, pmap, events := portmap.New(lhost + "." + lport)
        
        // For all port it map file contains entry then connect mapped ports and wait for new ones.
        if lport == "*" {
            for k, v := range pmap {
                if isAllowed(mapReader, pmap, k) {
                    go connect(opt, lhost, lport, rhost, passwd, interval, debug)
                }                    
            }
            
            // Wait for other ports 
            go procEvents(opt, lhost, rhost, passwd, interval, debug, events)
            
        } else {
            // If port mapping exist then connect using that else let ssh assign new random port
            if isAllowed(mapReader, pmap, lport) {
                go connect(opt, lhost, lport, rhost, passwd, interval, debug)
            }
        }
    })       
}