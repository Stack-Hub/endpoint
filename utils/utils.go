/* Copyright (C) Ashish Thakwani - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Ashish Thakwani <athakwani@gmail.com>, August 2017
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
    LOCK_SH     = syscall.LOCK_SH
    LOCK_EX     = syscall.LOCK_EX
    LOCK_NB     = syscall.LOCK_NB
    RUNPATH     = "/tmp/"
    SSHD_CONFIG = "/etc/ssh/sshd_config" 
    MATCHBLK    = `
Match User %s
    AllowTCPForwarding yes
    X11Forwarding no
    AllowAgentForwarding no
    PermitTTY yes
    AcceptEnv SSH_RFWD
    ForceCommand /usr/local/bin/trafficrouter -f $SSH_ORIGINAL_COMMAND
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
 * Convert int to bool
 */
func Itob(b int) bool {
    if b > 0 {
        return true
    }
    return false
 }



/*
 * Lock file
 */
func LockFile(filename string, truncate bool, how int) (*os.File, error) {

    mode := os.O_RDWR|os.O_CREATE
    
    if truncate {
        mode = mode|os.O_TRUNC
    }
    
    f, err := os.OpenFile(filename, mode, 0666)
    if err != nil {
        return nil, err
    }
    
    fd := f.Fd()
	err = unix.Flock(int(fd), how)
    if err != nil {
        f.Close()
        return nil, err
    }
    
    return f, nil
}

/*
 * Unlock file to unblock server
 */
func UnlockFile(f *os.File) error {
    fd := f.Fd()
    err := unix.Flock(int(fd), syscall.LOCK_UN)
    f.Sync()
    f.Close()
    return err
}
