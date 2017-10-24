/* Copyright (C) Ashish Thakwani - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Ashish Thakwani <athakwani@gmail.com>, August 2017
 */
package main

import (
    "os"
    "os/signal"
    "os/exec"
    "syscall"
    
    "github.com/duppercloud/trafficrouter/utils"
    "github.com/duppercloud/trafficrouter/dns"
    "github.com/duppercloud/trafficrouter/opt/require"
    "github.com/duppercloud/trafficrouter/opt/register"
    "github.com/duppercloud/trafficrouter/version"
    "github.com/prometheus/common/log"
//    reaper "github.com/ramr/go-reaper"
    "github.com/urfave/cli"
)
    
/*
 *  Cleanup before exit
 */
func cleanup() {    
    log.Debug("Cleaning up")
    register.Cleanup()
    require.Cleanup()
}


/*
 *  Install Signal handler for proper cleanup.
 */
func installHandler() {    
    sigs := make(chan os.Signal, 1)    
    signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
    go func() {
        sig := <-sigs
        log.Debug(sig)
        pgid, _ := syscall.Getpgid(syscall.Getpid())
        switch sig {
            case syscall.SIGHUP:
                syscall.Kill(-pgid, syscall.SIGHUP)
            case syscall.SIGTERM:
                syscall.Kill(-pgid, syscall.SIGTERM)
            case syscall.SIGINT:
                syscall.Kill(-pgid, syscall.SIGINT)
            case syscall.SIGQUIT:
                syscall.Kill(-pgid, syscall.SIGQUIT)
        }
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
 * Usecases: 

 * For connecting stateless service
 * trafficrouter --register app:80@lb
 * trafficrouter --require  app:80@localhost>80

 * For connecing stateful service
 * trafficrouter --register mysql:80@app
 * trafficrouter --require  mysql:80@localhost --on-connect 'MYSQL=$DIP:$DPORT service restart app'

 * For connecing service to itself
 * trafficrouter --register nodes:*@nodes
 * trafficrouter --require  nodes:*@localhost --on-connect 'NODESTRING=$NODESTRING:$DIP:$DPORT service restart mysql'

 * For stateful cluster (ex: mysql)
 * trafficrouter --register  manager:1186@nodes
 * trafficrouter --require   manager:1186@localhost

 * trafficrouter --register  manager:1186@mysql
 * trafficrouter --require   manager:1186@localhost

 * trafficrouter --register  nodes:*@manager
 * trafficrouter --require   nodes:*@localhost

 * trafficrouter --register  nodes:*@nodes
 * trafficrouter --require   nodes:*@localhost

 * trafficrouter --register  nodes:*@mysql
 * trafficrouter --require   nodes:*@localhost

 * trafficrouter --register  nodes:*@backup
 * trafficrouter --require   nodes:*@localhost

 */
func main() {

    /*
     * Install Signal handlers for proper cleanup.
     */
    installHandler()
    
    /*
     * Start zombie reaper in background
     */
    //go reaper.Reap()
    
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
	app.Usage = "Zero-config push based traffic router"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, D",
			Usage: "Enable debug logging",
		},
        cli.StringSliceFlag{
			Name:  "require, req",
            Usage: "Services required for application. Format `app:port[>laddr:lport]` e.g. db:3306 or app:8000>eth0:80",
		},	
        cli.StringSliceFlag{
			Name:  "register, reg",
            Usage: "Register this service. Format `app:port@raddr[:rport]` e.g. app:80@lb or app:80@lb:80 or app:80@lb:0",
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
        cli.StringFlag{
			Name:  "on-connect, oc",
			Usage: "execute command on service connect",
		},	
        cli.StringFlag{
			Name:  "on-disconnect, od",
			Usage: "execute command on service disconnect",
		},	
    }
    
    app.Before = func(c *cli.Context) error {
        
        debug := c.Bool("D")
		if debug {
            log.Base().SetLevel("debug")
		}

		return nil
	}

    app.Action = func (c *cli.Context) error {
        passwd := os.Getenv("PASSWD")
        if passwd == "" {
            passwd = "123456789"
        }

        // Set ulimit to max
        ulimit(999999)
        
        // Start local DNS server
        go dns.Start()

        // Poll specific values
        count := c.Int("count")
        interval := c.Int("interval")
        
        debug := c.Bool("D")
        
        for {        
            // Flag for go routing
            done := make(chan bool, 1)
            
            // Register services.
            register.Process(passwd, c.StringSlice("register"), count, interval, debug)
        
            // Wait for Needed service before registering.
            go require.Process(passwd, c.StringSlice("require"), func() {                
                
                for {
                    cmdargs := c.Args()
                    if len(cmdargs) > 0 {
                        cmd := cmdargs[0]
                        args := cmdargs[1:]
                        log.Debug("Executing ", cmd, args)
                        proc := exec.Command(cmd, args...)
                        proc.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

                        port := os.Getenv("PORT")
                        log.Debug("PORT=", port)
                        bindaddr := os.Getenv("BINDADDR")
                        log.Debug("BINDADDR=", bindaddr)

                        env := os.Environ()

/* TODO: Force applications to bind to localhost IPs
                        env = append(env, "FORCE_NET_VERBOSE=999")
                        env = append(env, "FORCE_NET_LOG='/var/log/bind.log'")
                        env = append(env, "FORCE_BIND_ADDRESS_V4=" + bindaddr)
                        env = append(env, "LD_PRELOAD=/usr/local/lib/force_bind.so")
*/
                        
                        if port == "*" {
                            env = append(env, "LD_PRELOAD=/usr/local/lib/listener.so")
                        }

                        proc.Env = env
                        log.Debug("env=", proc.Env)
                        proc.Stdout = os.Stdout
                        proc.Stderr = os.Stderr
                        err := proc.Start()
                        if err != nil {
                            log.Error(err)
                            os.Exit(1)
                        }

                        restart := make(chan os.Signal, 1)
                        signal.Notify(restart, syscall.SIGUSR1)

                        select {
                            // Restart on SIGUSR1
                            case sig := <-restart:
                                log.Debug(sig, " Restarting")
                                syscall.Kill(-proc.Process.Pid, syscall.SIGKILL)
                                continue
                            
                            case <-done:
                                log.Debug("Terminaing process")
                                syscall.Kill(-proc.Process.Pid, syscall.SIGKILL)
                        }
                    }
                    
                    break
                }

            })   

            reboot := make(chan os.Signal, 1)
            signal.Notify(reboot, syscall.SIGUSR2)

            // Reboot on SIGUSR2
            sig := <-reboot
            done <- true
            log.Debug(sig, " Rebooting.")
            cleanup()
        }
        
        utils.BlockForever()
        return nil        
    }
    
    if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}