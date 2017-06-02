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
package ssh

import (
    "os"
	"os/exec"
    "strings"
    
    "github.com/duppercloud/trafficrouter/utils"
    log "github.com/Sirupsen/logrus"
)

/*
 * Connection store
 */
var clients map[string]*exec.Cmd = make(map[string]*exec.Cmd, 1)

/*
 * Wait fo client to disconnect
 */
func wait(cmd *exec.Cmd, addr string){
    cmd.Wait()
    delete(clients, addr)
    log.Debug("Disconnected ", addr)
}

/*
 * SSH client connect
 */
func Connect(u string, pass string, ip string, lport string, rport string, debug bool) (string) {

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
                     "-t", 
                     "-o", "StrictHostkeyChecking=no", 
                     "-o", "UserKnownHostsFile=/dev/null", 
                     "-o", "SendEnv=SSH_RFWD", 
                     "-R", rport + ":localhost:" + lport, u + "@" + ip, 
                     "--",
                     isDebug(),
                     "{\"port\":" + lport + "}"}

    addr := u + "@" + ip
    log.Debug("Connecting ", addr)
    
    os.Setenv("LD_PRELOAD","/usr/lib/trafficrouter/rfwd.so")

	c := exec.Command(cmd, args...)    
    c.Stdout = os.Stdout
    c.Stderr = os.Stderr
    err := c.Start()
    utils.Check(err)
    
    var dynport string
    
    // Potential race condition, need to make sure ssh is connected before querying env
    for _, env := range c.Env {
        if strings.Contains(env, "SSHRFWD") {
            sshrfwd := strings.Split(env, "=")
            dynport = sshrfwd[1]
        }
    }
    
    //Add to Client store
    clients[addr] = c

    //Remove client when disconnected
    go wait(c, addr)
    
    return dynport
}

/*
 * Check if client is already connected
 */
func IsConnected(addr string) bool {
    ok := clients[addr]
    
    if ok != nil {
        return true
    }
    
    return false
}

/*
 * Diconnect client
 */
func Disconnect(addr string) {
    cmd := clients[addr]

    if cmd != nil {
        err := cmd.Process.Kill()
        utils.Check(err)        
    }
}
