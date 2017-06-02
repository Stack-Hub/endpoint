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
package utils

import (
    "fmt"
    "os"
    "syscall"
    
    "golang.org/x/sys/unix"
)

/*
 *  Default configurations
 */
const (
    RUNPATH     = "/tmp/"
    SSHD_CONFIG = "/etc/ssh/sshd_config" 
    MATCHBLK    = `
Match User %s
    AllowTCPForwarding yes
    X11Forwarding no
    AllowAgentForwarding no
    PermitTTY yes
    AcceptEnv SSH_RFWD
    ForceCommand /usr/sbin/trafficrouter -f $SSH_ORIGINAL_COMMAND
`
)


/*
 *  Config Struct is passed from client to server
 */
type Config struct {
    Port uint32 `json:"port"`
}

/*
 *  Host Struct is passed from forceCmd to server
 */
type Host struct {
    ListenPort  uint32 `json:"lisport"` // Localhost listening port of reverse tunnel
    RemoteIP    string `json:"raddr"`   // Remote IP 
    RemotePort  uint32 `json:"rport"`   // Port on which remote host connected
    Config      Config `json:"config"`  // Remote Config 
    Uid         int    `json:"uid"`     // User ID
    Uname       string `json:"uname"`   // Username
    Pid         int    `json:"pid"`     // Reverse tunnel process ID
}


/*
 *  Common error handling function.
 */
func Check(e error) {
    if e != nil {
        fmt.Fprintln(os.Stderr, e)
        panic(e)
    }
}

/*
 * Block Program Forever
 */
func BlockForever() {
    select {}
}



/*
 * Lock file
 */
func LockFile(filename string) int {

    f, err := os.Create(RUNPATH + filename)
    Check(err)
    
    fd := f.Fd()
	err = unix.Flock(int(fd), syscall.LOCK_EX)
    Check(err)
    
    return int(fd)
}

/*
 * Unlock file to unblock server
 */
func UnlockFile(fd int) {
	err := unix.Flock(fd, syscall.LOCK_UN)
    Check(err)
}
