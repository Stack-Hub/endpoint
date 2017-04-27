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
    
    "../../utils"
    "../../client"
    log "github.com/Sirupsen/logrus"
    "github.com/urfave/cli"
)

type callback func()

func parse(str string) (string, string, string, bool) {
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

func poll(lhost string, lport string, rhost string, passwd string, poll int, isDebug bool) {
    
    uname := lhost + "." + lport
    
    go func () {
        for {                    
            for i := 1; i<10; i++ {
                dns := rhost + strconv.Itoa(i)

                //Check if host exists
                log.Debug("Checking ", dns)
                raddr, err := net.LookupHost(dns)
                log.Debug(raddr)

                if err == nil && len(raddr) > 0 {
                    if !client.IsConnected(dns) {
                        cmd := client.Connect(uname, passwd, dns, lport, isDebug)
                        log.Debug(cmd)                    
                    }
                } 
            }    

            if poll <= 0 {
                return
            }
            time.Sleep( time.Duration(poll) * 1000 * time.Millisecond)
        }        
    }()
}

func connect(lhost string, lport string, rhost string, passwd string, isDebug bool) {
    uname := lhost + "." + lport
    cmd := client.Connect(uname, passwd, rhost, lport, isDebug)
    log.Debug(cmd)                                                    
}

func Process(c *cli.Context, cb callback) {
    isDebug := c.Bool("D")
    passwd := c.String("passwd")
    pollInt := c.Int("poll-interval")

    opts := c.String("register")
    if len(opts) > 0 {
        lhost, lport, rhost, wc := parse(opts)
        log.Println(lhost, lport, rhost, wc)

        if wc == true {
            poll(lhost, lport, rhost, passwd, pollInt, isDebug)
        } else {
            connect(lhost, lport, rhost, passwd, isDebug)      
        }
        
    }
        
    if cb != nil {
        cb()
    }
}