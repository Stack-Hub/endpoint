/* Copyright (C) Ashish Thakwani - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Ashish Thakwani <athakwani@gmail.com>, August 2017
 */
package utils

import (
    "fmt"
    "os"
    "os/exec"
    "net"
    "strconv"
    "io"
    "sync"

    "github.com/prometheus/common/log"
)


/*
 *  Config Struct is passed from client to server
 */
type Config struct {
    Port     uint32 `json:"port"`
    Instance uint32 `json:"instance"`
    Label    string `json:"label"`
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

type Endpoint struct {
	Host string
	Port uint32
}

func (endpoint *Endpoint) String() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}


/* 
 * CopyReadWriters copies biderectionally - output from a to b, and output of b into a. 
 * Calls the close function when unable to copy in either direction
 */
func CopyReadWriters(a, b io.ReadWriter, close func()) {
	var once sync.Once
	go func() {
		io.Copy(a, b)
		once.Do(close)
	}()

	go func() {
		io.Copy(b, a)
		once.Do(close)
	}()
}

/* 
 * Extract port from Address
 */
func GetHostPort(addr net.Addr) (host string, port int, err error) {
	host, portString, err := net.SplitHostPort(addr.String())
	if err != nil {
		return
	}
	port, err = strconv.Atoi(portString)
	return
}

/* 
 * Build TCPAddr from host & port
 */
func ParseTCPAddr(addr string, port uint32) (*net.TCPAddr, error) {
	if port == 0 || port > 65535 {
		return nil, fmt.Errorf("ssh: port number out of range: %d", port)
	}
	ip := net.ParseIP(string(addr))
	if ip == nil {
		return nil, fmt.Errorf("ssh: cannot parse IP address %q", addr)
	}
	return &net.TCPAddr{IP: ip, Port: int(port)}, nil
}


/*
* Get interface IP address
 */
func GetIP(iface string) (*net.IPAddr) {

    ip := []byte{127,0,0,1}

    if iface == "*" {
        ip = []byte{0,0,0,0}
    } else if len(iface) > 0 {
        ief, err := net.InterfaceByName(iface)

        if err == nil {
            addrs, err := ief.Addrs()
            if err == nil {
                ip = addrs[0].(*net.IPNet).IP        
            }
        }                
    }
    

    ipAddr := &net.IPAddr{
        IP: ip,
    }    
    
    return ipAddr
}

/*
 * Execute on-connect code
 */
func OnConnect(rhost string, rport string, lhost string, lport string, instance string, label string) {
    if _, err := os.Stat("/var/lib/dupper/onconnect"); !os.IsNotExist(err) {
        cmd := "bash"
        args := []string{"/var/lib/dupper/onconnect",}

        c := exec.Command(cmd, args...)
        env := os.Environ()
        env = append(env, fmt.Sprintf("INSTANCE=%s", instance))
        env = append(env, fmt.Sprintf("LABEL=%s", label))
        env = append(env, fmt.Sprintf("REMOTEHOST=%s", rhost))
        env = append(env, fmt.Sprintf("REMOTEPORT=%s", rport))
        env = append(env, fmt.Sprintf("LOCALHOST=%s", lhost))
        env = append(env, fmt.Sprintf("LOCALPORT=%s", lport))
        c.Env = env
        c.Stdout = os.Stdout
        c.Stderr = os.Stderr
        err = c.Run()
        if err != nil {
            log.Error(err)
        }
    }        
}

/*
 * Execute on-disconnect code
 */
func OnDisconnect(rhost string, rport string, lhost string, lport string, instance string, label string) {
    if _, err := os.Stat("/var/lib/dupper/ondisconnect"); !os.IsNotExist(err) {
        cmd := "bash"
        args := []string{"/var/lib/dupper/ondisconnect",}

        c := exec.Command(cmd, args...)
        env := os.Environ()
        env = append(env, fmt.Sprintf("INSTANCE=%s", instance))
        env = append(env, fmt.Sprintf("LABEL=%s", label))
        env = append(env, fmt.Sprintf("REMOTEHOST=%s", rhost))
        env = append(env, fmt.Sprintf("REMOTEPORT=%s", rport))
        env = append(env, fmt.Sprintf("LOCALHOST=%s", lhost))
        env = append(env, fmt.Sprintf("LOCALPORT=%s", lport))
        c.Env = env
        c.Stdout = os.Stdout
        c.Stderr = os.Stderr
        err = c.Run()
        if err != nil {
            log.Error(err)
        }
    }
}
