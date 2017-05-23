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
    var expr = regexp.MustCompile(`^(.+):([0-9]+)@([^:\*]+)$`)
	parts := expr.FindStringSubmatch(str)
	if len(str) == 0 {
        utils.Check(errors.New(fmt.Sprintf("Option parse error: [%s]. Format lhost:lport@rhost?\n", str)))
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
 *  Connect to remote host and periodically check the state.
 */
func connect(opt string, lhost string, lport string, rhost string, 
             passwd string, interval int, debug bool) {

    uname := lhost + "." + lport
    for {                  
        
        //Check if host exists
        ipArr, _ := net.LookupHost(rhost)
        
        for _, ip := range ipArr {
            addr := uname + "@" + ip

            if !ssh.IsConnected(addr) {
                fmt.Println("Connecting...", addr)
                ssh.Connect(uname, passwd, ip, lport, debug)
            }            
        }

        time.Sleep( time.Duration(interval) * 1000 * time.Millisecond)
    }        
}

/*
 *  Process --regiser options
 */
func Process(passwd string, opts []string, count int, interval int, debug bool) {
    log.Debug(opts)

    forEach(opts, func(opt string, lhost string, lport string, rhost string) {
        go connect(opt, lhost, lport, rhost, passwd, interval, debug)      
    })        
}