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
package client

import (
    "os"
	"os/exec"
    
    "github.com/duppercloud/trafficrouter/utils"
    log "github.com/Sirupsen/logrus"
)

var clients map[string]*exec.Cmd = make(map[string]*exec.Cmd, 1)

func wait(cmd *exec.Cmd, addr string){
    cmd.Wait()
    delete(clients, addr)
    log.Debug("Disconnected ", addr)
}

func Connect(u string, pass string, h string, p string, debug bool) *exec.Cmd {

    isDebug := func() string {
        if debug == true {
            return "-D"
        }
        return ""
    }
	// ssh open reverse tunnel
	cmd := "sshpass"
	args := []string{"-p", pass,
                     "ssh",
                     "-q", 
                     "-t", 
                     "-o", "StrictHostkeyChecking=no", 
                     "-o", "UserKnownHostsFile=/dev/null", 
                     "-R", "0:localhost:" + p, u + "@" + h, 
                     "--",
                     isDebug(),
                     "{\"port\":" + p + "}"}

    addr := u + "@" + h
    log.Debug("Connecting ", addr)
    
	c := exec.Command(cmd, args...)
    c.Stdout = os.Stdout
    c.Stderr = os.Stderr
    err := c.Start()
    utils.Check(err)
    
    //Add to Client store
    clients[addr] = c

    //Remove client when disconnected
    go wait(c, addr)
    
    return c
}

func IsConnected(addr string) bool {
    ok := clients[addr]
    
    if ok != nil {
        return true
    }
    
    return false
}

func Disconnect(cmd *exec.Cmd) {
 
    err := cmd.Process.Kill()
    utils.Check(err)
}
