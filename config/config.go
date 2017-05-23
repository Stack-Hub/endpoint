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
package config

import (
    "fmt"
    "encoding/json"
    "os"
    "os/user"
    "net"
    "strconv"
    "errors"
    "regexp"

    "github.com/duppercloud/trafficrouter/utils"
    log "github.com/Sirupsen/logrus"    
)

var fd int

/*
 * Cleanup on termination.
 */
func Cleanup() {
    utils.UnlockFile(fd)    
}


/*
 *  SSH_CLIENT parse logic
 */
func parse(str string) (string, string, string) {
    var expr = regexp.MustCompile(`^(.*) (.*) (.*)$`)
	parts := expr.FindStringSubmatch(str)
	if len(str) == 0 {
        utils.Check(errors.New(fmt.Sprintf("SSH_CLIENT parse error: [%s]\n", str)))
	}
            
    return parts[1], parts[2], parts[3]
}


/*
 *  Get tunnel connections parameters in host struct
 */
func connInfo(h *utils.Host) {
    // Get SSH reverse tunnel connection information from environment.    

    sshClient := os.Getenv("SSH_CLIENT")
    remoteHost, remotePort, _ := parse(sshClient)
    rPort, _ := strconv.Atoi(remotePort)
    h.RemoteIP = remoteHost
    h.RemotePort = uint32(rPort)
    log.Debug("SSH_CLIENT", sshClient)
    
    lisPort, _ := strconv.Atoi(os.Getenv("SSH_RFWD"))
    log.Debug("SSH_RFWD", os.Getenv("SSH_RFWD"))

    h.ListenPort = uint32(lisPort)
}


/*
 *  Get Client configuration parameters in host struct
 */
func config(h *utils.Host) {
    
    // Get Client config which should be the last argument
    cfgstr := os.Args[len(os.Args) - 1]
    log.Debug(cfgstr)
    
    // Conver config to json
    cfg := utils.Config{}
    json.Unmarshal([]byte(cfgstr), &cfg)
    
    // Update and log host var
    h.Config = cfg
    h.Uid = os.Getuid()
    
    u, err := user.Current()
    utils.Check(err)
    h.Uname = u.Username
    
}

/*
 *  Send host struct to uname socket
 */
func write(uname string, h *utils.Host) {

    // Form Unix socket based on pid 
    f := utils.RUNPATH + uname + ".sock"
    log.Debug("SOCK: ", f)
    c, err := net.Dial("unix", f)
    utils.Check(err)
    
    defer c.Close()

    // Convert host var to json and send to server
    payload, err := json.Marshal(h)
    utils.Check(err)
    
    // Send to server over unix socket.
    _, err = c.Write(payload)
    utils.Check(err)
}

/*
 *  Get connection information of ssh tunnel and send the 
 *  information to server.
 */
func Send() {
    
    // Get parent proc ID which will be flock's pid.
    ppid := os.Getppid()
    log.Debug("ppid = ", ppid)

    // Flock on pid file
    fd = utils.LockFile(ppid)
    
    //Host to store connection information
    var h utils.Host
    h.Pid = ppid
    
    //Get socket connection parameters in host struct
    connInfo(&h)
    
    //Get client config parameters in host struct
    config(&h)

    //Log complete host struct.
    log.Debug(h)
    fmt.Println("Connected on port", h.RemotePort)
    
    // Get Current user
    u, _ := user.Current()

    //Send host struct to user name unix socket.
    write(u.Username, &h)
    
    // Wait for Interrupt or Parent exit
    wait(fd)

}
