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

func Process(c *cli.Context, cb callback) {
    
    isDebug := c.Bool("D")

    // Get password
    passwd := c.String("passwd")

    opts := c.String("register")
    if len(opts) > 0 {
        lhost, lport, rhost, wc := parse(opts)

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
                    cmd := client.Connect(uname, passwd, rdns, lport, isDebug)
                    log.Debug(cmd)          
                    i++
                } 

                poll := c.Int("poll-interval")
                if poll <= 0 {
                    break
                }

                time.Sleep( time.Duration(poll) * 1000 * time.Millisecond)
            }
        } else {
            cmd := client.Connect(uname, passwd, rhost, lport, isDebug)
            log.Debug(cmd)                                                    
        }
        
    }
        
    if cb != nil {
        cb()
    }
}