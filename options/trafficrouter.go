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
package main

import (
    "fmt"
    "net"
    "os"
    "io"
    "strconv"
    "regexp"
    "errors"
    "syscall"
    "strings"
    "os/exec"
    "os/signal"
    "time"
    "bufio"
    
    "./omap"
    "./client"
    "./forcecmd"
    "./user"
    "./server"
    "./utils"
    "./options/require"
    log "github.com/Sirupsen/logrus"
    "github.com/urfave/cli"
    "github.com/kardianos/osext"
)

var m * omap.OMap

/*
 *  Cleanup before exit
 */
func cleanup() {    
    log.Debug("Cleaning up")
    forcecmd.Cleanup()
    user.Cleanup()
}

/*
 *  Install Signal handler for proper cleanup.
 */
func installHandler() {
    sigs := make(chan os.Signal, 1)    
    signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        sig := <-sigs
        log.Debug(sig)
        cleanup()
        os.Exit(1)
    }()    
}


func parseRegister(str string) (string, string, string, bool) {
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


func TrafficRouter(c *cli.Context) error {

    /*
     * Install Signal handlers for proper cleanup.
     */
    installHandler()
    
    /*
     * Global error handling.
     * Cleanup and exit.
     */
    defer func() {
        if r := recover(); r != nil {
            log.Debug("Recovered ", r)
            cleanup()
            os.Exit(1)
        }
    }()    
    
    ch = make(chan bool, 2)
    registered = false
    
    // Send message for Force Command mode and return.
    if (c.Bool("f") == true) {
        forcecmd.SendConfig()
        return nil
    }

    isDebug := c.Bool("D")
    
    passwd := c.String("passwd")
    if passwd == "" {
        log.Fatal("Empty password. Please provide password with --passwd option")
    }
        
    // Wait for Needed service before registering.
    if (c.String("require") != "") {
        require.process(c.String("require"))
    }
    
    /* Client mode */
    if (c.String("register") != "") {
        lhost, lport, rhost, wc := parseRegister(c.String("register"))

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
                    defer client.Disconnect(cmd)
                    log.Debug(cmd)          
                    i++
                } 
                poll := c.Int("poll-interval")
                time.Sleep( time.Duration(poll) * 1000 * time.Millisecond)
            }
        } else {
            cmd := client.Connect(uname, passwd, rhost, lport, isDebug)
            defer client.Disconnect(cmd)
            log.Debug(cmd)                                                    
        }

        utils.BlockForever()
    }         
    
    if (c.String("cmd") != "") {
        cmdargs := strings.Split(c.String("cmd")," ")
        cmd := cmdargs[0]
        args := cmdargs[1:]
        log.Debug("Executing ", cmd, args)
        c := exec.Command(cmd, args...)
        stdout, _ := c.StdoutPipe()
        stderr, _ := c.StderrPipe()
        
        stdoutscanner := bufio.NewScanner(stdout)
        stderrscanner := bufio.NewScanner(stderr)
        go func() {
            for stdoutscanner.Scan() {
                fmt.Println(stdoutscanner.Text())
            }
        }()
        go func() {
            for stderrscanner.Scan() {
                fmt.Println(stderrscanner.Text())
            }
        }()

        
        c.Start()
    }
    
    utils.BlockForever()
    return nil
}