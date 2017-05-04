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
    "os"
    "os/signal"
    "os/exec"
    "syscall"
    
    "github.com/duppercloud/trafficrouter/monitor"
    "github.com/duppercloud/trafficrouter/config"
    "github.com/duppercloud/trafficrouter/user"
    "github.com/duppercloud/trafficrouter/utils"
    "github.com/duppercloud/trafficrouter/opt/require"
    "github.com/duppercloud/trafficrouter/opt/register"
    "github.com/duppercloud/trafficrouter/version"
    log "github.com/Sirupsen/logrus"
    "github.com/urfave/cli"
)
    
/*
 *  Cleanup before exit
 */
func cleanup() {    
    log.Debug("Cleaning up")
    config.Cleanup()
    server.Cleanup()
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

/*
 * Increase ulimit to handle large concurrent connections.
 */
func ulimit(num uint64){
    var rLimit syscall.Rlimit
    err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
    if err != nil {
        log.Debug("Error Getting Rlimit ", err)
    }
    log.Debug("rlimit=", rLimit)
    rLimit.Max = num
    rLimit.Cur = num
    err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
    if err != nil {
        log.Debug("Error Setting Rlimit ", err)
    }
    err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
    if err != nil {
        log.Debug("Error Getting Rlimit ", err)
    }
    log.Debug("Rlimit Final", rLimit)
}

/*
 * Entrypoint
 */
func main() {

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

    
    app := cli.NewApp()
	app.Name = "trafficrouter"
	app.Version = version.FullVersion()
	app.Author = "@athakwani"
	app.Email = "athakwani@gmail.com"
	app.Usage = "Zero-config push based load balancer"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, D",
			Usage: "Enable debug logging",
		},
        cli.StringFlag{
			Name:  "passwd, p",
			Usage: "Password to secure connections",
			Value: "123456789",
		},	
        cli.StringSliceFlag{
			Name:  "require, req",
            Usage: "Services required for application. Format `app:port@laddr[:lport]` e.g. db:3306@localhost",
		},	
        cli.StringSliceFlag{
			Name:  "register, reg",
            Usage: "Register this service. Format `app:port@raddr` e.g. app:80@lb or app:80@lb-*",
		},	
        cli.BoolFlag{
			Name:  "forcecmd, f",
            Usage: "SSH force command (used internally)",
			Hidden: false,
		},	
        cli.IntFlag{
			Name:  "count, c",
            Usage: "Wildcard count",
			Value: 10,
		},	
        cli.IntFlag{
			Name:  "interval, i",
            Usage: "Interval to detect new hosts, used with --register for wildcard option",
			Value: 10,
		},	
    }
    
    app.Before = func(c *cli.Context) error {
        
        debug := c.Bool("D")
		if debug {
			log.SetLevel(log.DebugLevel)
		}

		return nil
	}

    app.Action = func (c *cli.Context) error {
        // Send message for Force Command mode and return.
        if (c.Bool("f") == true) {
            config.Send()
            return nil
        }

        passwd := c.String("passwd")
        if passwd == "" {
            log.Fatal("Empty password. Please provide password with --passwd option")
        }

        // Set ulimit to max
        ulimit(999999)
        
        // Poll specific values
        count := c.Int("count")
        interval := c.Int("interval")
        
        debug := c.Bool("D")
        // Wait for Needed service before registering.
        go require.Process(passwd, c.StringSlice("require"), func() {

            // Register services.
            register.Process(passwd, c.StringSlice("register"), count, interval, debug)

            cmdargs := c.Args()
            if len(cmdargs) > 0 {
                cmd := cmdargs[0]
                args := cmdargs[1:]
                log.Debug("Executing ", cmd, args)
                proc := exec.Command(cmd, args...)
                proc.Stdout = os.Stdout
                proc.Stderr = os.Stderr
                proc.Run()
            }
        })    

        utils.BlockForever()
        return nil        
    }
    
    if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}