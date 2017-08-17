/* Copyright (C) Ashish Thakwani - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Ashish Thakwani <athakwani@gmail.com>, August 2017
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
func wait(cmd *exec.Cmd, hash string){
    cmd.Wait()
    delete(clients, hash)
    log.Debug("Disconnected ", hash)
}

/*
 * SSH client connect
 */
func Connect(u string, pass string, ip string, lport string, rport string, hash string, debug bool) (string) {

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
                     "-R", rport + ":localhost:" + lport, 
                     u + "@" + ip, 
                     "--",
                     isDebug(),
                     "{\"port\":" + lport + "}"}

    log.Debug("Connecting ", hash)
    
    if rport == "0" {
        os.Setenv("LD_PRELOAD","/usr/local/lib/rfwd.so")
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
    clients[hash] = c

    //Remove client when disconnected
    go wait(c, hash)
    

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
func IsConnected(hash string) bool {
    ok := clients[hash]
    
    if ok != nil {
        return true
    }
    
    return false
}

/*
 * Diconnect client
 */
func Disconnect(hash string) {
    cmd := clients[hash]

    if cmd != nil {
        err := cmd.Process.Kill()
        utils.Check(err)        
    }
}
