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
    "strconv"
    "io"
    "bufio"
    "fmt"
    
    "github.com/pipecloud/endpoint/utils"
    "github.com/pipecloud/endpoint/dns"
    "github.com/pipecloud/endpoint/opt/require"
    "github.com/pipecloud/endpoint/opt/register"
    "github.com/pipecloud/endpoint/version"
    "github.com/prometheus/common/log"
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
	app.Name = "endpoint"
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
        dns.Start()
        port := os.Getenv("PORT")
        log.Debug("PORT=", port)
        instance, _ := strconv.Atoi(os.Getenv("INSTANCE"))
        bindaddr := dns.GenerateIP(uint32(instance)).String()
        log.Debug("BINDADDR=", bindaddr)
        os.Setenv("BINDADDR", bindaddr)

        // Poll specific values
        interval := c.Int("interval")
        
        debug := c.Bool("D")
        
        for {        
            // Flag for go routing
            done := make(chan bool, 1)
            
            // Register services.
            register.Process(passwd, c.StringSlice("register"), interval, debug)
        
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

                        env := os.Environ()

                        if port == "*" {
                            env = append(env, "LD_PRELOAD=/usr/local/lib/listener.so")
                        }

                        proc.Env = env
                        log.Debug("env=", proc.Env)

                        stream := func (prefix string, out io.Reader) {
                            scanner := bufio.NewScanner(out)
                            for scanner.Scan() {
                                m := scanner.Text()
                                fmt.Println(prefix, m)
                            }
                            
                        }

                        
                        stdout, err := proc.StdoutPipe()
                        if err != nil {
                            os.Exit(1)
                        }                        
                        
                        go stream("stdout:", stdout)
                        stderr, err := proc.StderrPipe()
                        if err != nil {
                            os.Exit(1)
                        }                        

                        go stream("stderr:", stderr)
                        
                        err = proc.Start()
                        if err != nil {
                            log.Error(err)
                            os.Exit(1)
                        }
                        
                        go func(){
                            proc.Wait()
                            fmt.Println("Process Terminated")
                            register.Cleanup()
                        }()

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