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
package forcecmd

import (
    "encoding/json"
    "os"
    "os/user"
    "net"

    "../utils"
    netutil "github.com/shirou/gopsutil/net"
    ps "github.com/shirou/gopsutil/process"
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
 *  Get Parent process's pid and commandline.
 */
func getProcParam(p int32) (*ps.Process, string) {

    // Init process struct based on PID
    proc, err := ps.NewProcess(p)
    utils.Check(err)

    // Get command line of parent process.
    cmd, err := proc.Cmdline()
    utils.Check(err)
    
    return proc, cmd
}

/*
 *  Get tunnel connections parameters in host struct
 */
func getConnParams(pid int32, h *utils.Host) {
    // Get SSH reverse tunnel connection information.
    // 3 sockers are opened by ssh:
    // 1. Connection from client to server
    // 2. Listening socket for IPv4
    // 3. Listening socket for IPv6
    conns, err := netutil.ConnectionsPid("inet", pid)
    utils.Check(err)
    log.Debug(conns)

    for _, c := range conns {
        // Family = 2 indicates IPv4 socket. Store Listen Port
        // in host structure.
        if c.Family == 2 && c.Status == "LISTEN" {
            h.ListenPort = c.Laddr.Port
        }

        // Store Established connection IP & Port in host structure.
        if c.Family == 2 && c.Status == "ESTABLISHED" {
            h.RemoteIP   = c.Raddr.IP
            h.RemotePort = c.Raddr.Port
        }
    }
}


/*
 *  Get Client configuration parameters in host struct
 */
func getConfigParams(h *utils.Host) {
    
    // Get Client config which should be the last argument
    cfgstr := os.Args[len(os.Args) - 1]
    log.Debug(cfgstr)
    
    // Conver config to json
    cfg := utils.Config{}
    json.Unmarshal([]byte(cfgstr), &cfg)
    
    // Update and log host var
    h.AppPort = cfg.Port                
    h.Config = cfg
    h.Uid = os.Getuid()
    
}

/*
 *  Send host struct to uname socket
 */
func writeHost(uname string, h *utils.Host) {

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
func SendConfig() {
    
    // Get parent proc ID which will be flock's pid.
    ppid := os.Getppid()
    log.Debug("ppid = ", ppid)

    // Flock on pid file
    fd = utils.LockFile(ppid)

    // Get parent process params
    _, pcmd := getProcParam(int32(ppid))
    log.Debug("Parent Process cmdline = ", pcmd)
    
    //Host to store connection information
    var h utils.Host
    h.Pid = ppid
    
    //Get socket connection parameters in host struct
    getConnParams(int32(ppid), &h)
    
    //Get client config parameters in host struct
    getConfigParams(&h)

    //Log complete host struct.
    log.Println(h)
    
    // Get Current user
    u, _ := user.Current()

    //Send host struct to user name unix socket.
    writeHost(u.Username, &h)
    
    // Wait for Interrupt or Parent exit
    waitForExit(fd)

}
