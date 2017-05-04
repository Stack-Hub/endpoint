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
    "strconv"
    "time"
    
    "github.com/duppercloud/trafficrouter/utils"
    "github.com/duppercloud/trafficrouter/ssh"
    log "github.com/Sirupsen/logrus"
)

/*
 *  forEach parser callback
 */
type parsecb func(string, string, string, string, string)

/*
 *  --regiser option parser logic
 */
func parse(str string) (string, string, string, string) {
    var expr = regexp.MustCompile(`^(.+):([0-9]+)@([^:\*]+)(\*)?$`)
	parts := expr.FindStringSubmatch(str)
	if len(str) == 0 {
        utils.Check(errors.New(fmt.Sprintf("Option parse error: [%s]. Format lhost:lport@rhost(*)?\n", str)))
	}
            
    return parts[1], parts[2], parts[3], parts[4]
}

/*
 *  -regiser options iterater
 */
func forEach(opts []string, cb parsecb) {
    for _, opt := range opts {
        lhost, lport, rhost, wc := parse(opt)

        log.Debug("laddr=", lhost, ",",
                  "lport=", lport, ",",
                  "rhost=", rhost, ",",
                  "wc=",    wc)
        
        cb(opt, lhost, lport, rhost, wc)
    }
}

/*
 *  Periodic poll for remote host to detect the presence
 *  and connect when available.
 */
func poll( opt string, lhost string, lport string, rhost string, 
           passwd string, interval int, count int, debug bool) {
    
    uname := lhost + "." + lport
    for {                    
        log.Debug("Resolving...", opt)
        for i := 1; i < count; i++ {
            dns := rhost + strconv.Itoa(i)
            
            //Check if host exists
            raddr, err := net.LookupHost(dns)

            if err == nil && len(raddr) > 0 {
                addr := uname + "@" + dns

                if !ssh.IsConnected(addr) {
                    ssh.Connect(uname, passwd, dns, lport, debug)
                }
            } 
        }    

        if interval <= 0 {
            return
        }
        
        time.Sleep( time.Duration(interval) * 1000 * time.Millisecond)
    }        
}

/*
 *  Connect to remote host and periodically check the state.
 */
func connect(opt string, lhost string, lport string, rhost string, 
             passwd string, interval int, debug bool) {

    uname := lhost + "." + lport
    for {                  
        addr := uname + "@" + rhost
        if !ssh.IsConnected(addr) {
            fmt.Println("Connecting...", opt)
            ssh.Connect(uname, passwd, rhost, lport, debug)
        }
        time.Sleep( time.Duration(interval) * 1000 * time.Millisecond)
    }        
}

/*
 *  Process --regiser options
 */
func Process(passwd string, opts []string, count int, interval int, debug bool) {
    log.Debug(opts)

    forEach(opts, func(opt string, lhost string, lport string, rhost string, wildcard string) {
        if wildcard == "*" {
            go poll(opt, lhost, lport, rhost, passwd, interval, count, debug)
        } else {
            go connect(opt, lhost, lport, rhost, passwd, interval, debug)      
        }        
    })        
}