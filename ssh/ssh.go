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
    "fmt"
    "os"
	"os/exec"
    "bufio"
    "strconv"
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
    
    if rport == "0" {
        os.Setenv("LD_PRELOAD","/usr/lib/trafficrouter/rfwd.so")
    } else {
        os.Setenv("SSH_RFWD",rport)
    }

	c := exec.Command(cmd, args...)    

    cmdReader, err := c.StderrPipe()
    utils.Check(err)

    scanner := bufio.NewScanner(cmdReader)
    
    c.Stdout = os.Stdout
    err = c.Start()
    utils.Check(err)

    //Add to Client store
    clients[addr] = c

    //Remove client when disconnected
    go wait(c, addr)
    

    if rport == "0" {    
        var dynport int

        for scanner.Scan() {
            output := scanner.Text()
            if strings.Contains(output, "Connection refused") {
                break
            }

            num, _ := fmt.Sscanf(output, "Allocated port %d for remote forward", &dynport)
            log.Debug(scanner.Text())
            if num == 1 {
                break
            }
        }

        log.Debug("dynport=", dynport)

        return strconv.Itoa(dynport)
    }
    
    return rport
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
