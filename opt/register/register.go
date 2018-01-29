/* Copyright (C) Ashish Thakwani - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Ashish Thakwani <athakwani@gmail.com>, August 2017
 */
package register

import (
    "fmt"
    "net"
    "regexp"
    "errors"
    "time"
    "net/rpc"
    "net/http"
    "strconv"
    
    "github.com/duppercloud/trafficrouter/utils"
    "github.com/duppercloud/trafficrouter/client"
    "github.com/prometheus/common/log"
    "github.com/bogdanovich/dns_resolver"
    netstat "github.com/shirou/gopsutil/net"
)

/*
 *  opt struct
 */
type Reg struct {
    opt        string               //option string
    lhost      string               //local hostname
    lport      uint32               //local port to connect
    operator   string               //port function operator
    rhost      string               //remote host to connect to
    rport      uint32               //remote host port
    user       string                //remote username
}

var goroutines map[string]chan bool = make(map[string]chan bool, 1)
var rpcRegistered bool = false

type Args struct {
    Lport uint32
    Rport uint32
}

type RPC struct {
    opts []string
    passwd   string
    interval int
    debug    bool
}

/*
 *  forEach parser callback
 */
type parsecb func(*Reg) error

func Cleanup() {
    fmt.Println("Register: Closing all connections.")
    for key, done := range goroutines {
        fmt.Println("Register: Closing done=", done)
        if done != nil {
            close(done)
            delete(goroutines, key)
        }
    }
}

func lookupHost(rhost string) ([]net.IP, error) {
    resolver, err := dns_resolver.NewFromResolvConf("/etc/resolv.conf")
    if err != nil {
        log.Error(err)
        return nil, err
    }
    
    return resolver.LookupHost(rhost)
}

/*
 *  --Regiser option parser logic
 */
func parse(str string) (string, uint32, string, uint32) {
    var expr = regexp.MustCompile(`([a-zA-Z^:][a-zA-Z0-9\-\.]+):([0-9]+|\*)(@([^:]+)(:([0-9]+))?)$`)
	parts := expr.FindStringSubmatch(str)

    if len(parts) == 0 {
        utils.Check(errors.New(fmt.Sprintf("Option parse error: [%s]. Format lhost:lport[@rhost:rport]\n", str)))
	}
    
    if parts[2] == "*" {
        parts[2] = "0"
    }
    
    lport, err := strconv.Atoi(parts[2])
    if err != nil {
        utils.Check(errors.New(fmt.Sprintf("Option parse error: [%s]. Format lhost:lport[@rhost:rport]\n", str)))
    }
    
    if parts[6] == "" {
        parts[6] = parts[2]
    }

    rport, err := strconv.Atoi(parts[6])
    if err != nil {
        utils.Check(errors.New(fmt.Sprintf("Option parse error: [%s]. Format lhost:lport[@rhost:rport]\n", str)))
    }
    
    
    return parts[1], uint32(lport), parts[4], uint32(rport)
}

/*
 *  -regiser options iterater
 */
func forEach(opts []string, cb parsecb) error {
    for _, opt := range opts {

        r := Reg{opt: opt}
        r.lhost, r.lport, r.rhost, r.rport = parse(opt)

        if r.lport == 0 {
            r.user = r.lhost
        } else {
            r.user = r.lhost + "." + fmt.Sprint(r.lport)
        }

        
        log.Debug("lhost=", r.lhost, ",",
                  "lport=", r.lport, ",",
                  "rhost=", r.rhost, ",",
                  "rport=", r.rport, ",",
                  "ruser=", r.user)
        
        if cb != nil {
            if err := cb(&r); err != nil {
                return err
            }
        }
    }
    return nil
}


func (r Reg) reconnect(passwd string, interval int, debug bool) {
    // Channel to notify when to stop this go routine
    done := make(chan bool)
    goroutines[r.rhost] = done

    for {      
        
        // Go connect, ignore errors and keep retrying
        go r.connect(passwd, debug)
        
        // Diconnect all ssh connection if channel is closed and return.
        select {
            case _, ok := <- done:
                log.Debug("Terminating goroutine")
                if !ok {
                    // Check if host exists
                    ipArr, _ := lookupHost(r.rhost)

                    for _, ip := range ipArr {
                        hash := r.lhost + "." + fmt.Sprint(r.lport) + "@" + ip.String()
                        client.Disconnect(hash)
                    }
                    return
                }
            case  <- time.After( time.Duration(interval) * 1000 * time.Millisecond):

            /* no-op */
        }
    }
}


func (r Reg) isPortOpen() bool {
    conns, err := netstat.Connections("inet")
    if err != nil {
        log.Debug(err)
        return false
    }

    for _, conn := range conns {
        if r.lport !=0 && conn.Laddr.Port == r.lport {
            return true
        }
    }            
    
    return false
}

/*
 *  Connect internal to remote host and periodically check the state.
 */
func (r Reg) connect(passwd string, debug bool) error {

    if r.isPortOpen() {
        
        ipArr, err := lookupHost(r.rhost)
        if err != nil {
            log.Error(err)
            return err
        }
        
        // Connect to all IP address for remote host
        for _, ip := range ipArr {
            hash := r.lhost + "." + fmt.Sprint(r.lport) + "@" + ip.String()

            // flag for skipping self connection
            skip := false

            // Make sure to not connect to itself for container:* scenario
            laddrs, _ := net.InterfaceAddrs()
            for _, address := range laddrs {
                if ipnet, ok := address.(*net.IPNet); ok {
                    if ipnet.IP.To4() != nil {
                        if ip.String() == ipnet.IP.String() {
                            skip = true
                            break
                        }
                    }
                }
            }

            // Skip if remote IP is one of local interface ip
            if skip {
                continue 
            }

            // connect to dynamic port.
            // store assigned port in map
            // Use the same port for rest of the connections.
            if !client.IsConnected(hash) {
                fmt.Println("Connecting...", hash)
                err = client.Connect(r.user, passwd, ip.String(), r.lport, r.rport, hash, debug)
                if err != nil {
                    return err
                }
            }
        }
    }

    return nil
}        

/*
 *  Connect to remote host and periodically check the state.
 */
func (r Reg) Connect(passwd string, interval int, debug bool) error {

    err := r.connect(passwd, debug)
    go r.reconnect(passwd, interval, debug)
    return err
}

/*
 *  Connect/Disconnect changes based on portmap
 */
func (r Reg) Disconnect() {
    // Disconnect all connections for lport by closing goroutine channel.
    if goroutines[r.rhost] != nil {
        close(goroutines[r.rhost])
        delete(goroutines, r.rhost)                
    }
}


/*
 *  Connect to all hosts
 */
func (_rpc RPC) Connect(args *Args, errno *int) error {
    
    log.Debug("RPC Connect invoked with args=", args)
    // Start event loop for each option
    forEach(_rpc.opts, func(r *Reg) error {
        if r.lport == 0 {
            rDynamic := r
            rDynamic.lport = args.Lport
            rDynamic.rport = args.Rport
            rDynamic.Connect(_rpc.passwd, _rpc.interval, _rpc.debug)            
        }
        return nil
    })    
        
    *errno = 0
    return nil
}

/*
 *  Connect to all hosts
 */
func (_rpc RPC) Disconnect(args *Args, errno *int) error {    
    log.Debug("RPC Disconnect invoked with args=", args)
    // Start event loop for each option
    forEach(_rpc.opts, func(r *Reg) error {
        r.lport = args.Lport
        r.rport = args.Rport
        r.Disconnect()
        return nil
    })    
        
    *errno = 0
    return nil
}

/*
 *  Process --regiser options
 */
func Process(passwd string, opts []string, interval int, debug bool) {
    log.Debug(opts)

    if !rpcRegistered {
        // Init RPC struct and export for remote calling
        _rpc := new(RPC)
        _rpc.opts = opts
        _rpc.passwd = passwd
        _rpc.interval = interval
        _rpc.debug = debug

        rpc.Register(_rpc)
        rpc.HandleHTTP()
        l, e := net.Listen("tcp", "localhost:3877")
        if e != nil {
            log.Error("listen error:", e)
        }
        go http.Serve(l, nil)
        rpcRegistered = true
    }

    // Start event loop for each option
    forEach(opts, func(r *Reg) error {
        if r.lport != 0 {
            if err := r.Connect(passwd, interval, debug); err != nil{
                log.Error(err)
            }
        }
        return nil
    })
}



