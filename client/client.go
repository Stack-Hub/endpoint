/* Copyright (C) Ashish Thakwani - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Ashish Thakwani <athakwani@gmail.com>, October 2017
 */
package client

import (
	"fmt"
	"io"
	"net"
    "os"

	"golang.org/x/crypto/ssh"
    "github.com/duppercloud/trafficrouter/utils"
    log "github.com/Sirupsen/logrus"
)


/*
 * Connection store
 */
var clients map[string]net.Listener = make(map[string]net.Listener, 1)


type Endpoint struct {
	Host string
	Port uint32
}

func (endpoint *Endpoint) String() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

func handleClient(client net.Conn, remote net.Conn) {
	defer client.Close()
	chDone := make(chan bool)

	// Start remote -> local data transfer
	go func() {
		_, err := io.Copy(client, remote)
		if err != nil {
			log.Println(fmt.Sprintf("error while copy remote->local: %s", err))
		}
		chDone <- true
	}()

	// Start local -> remote data transfer
	go func() {
		_, err := io.Copy(remote, client)
		if err != nil {
			log.Println(fmt.Sprintf("error while copy local->remote: %s", err))
		}
		chDone <- true
	}()

	<-chDone
}

func Connect(u string, pass string, rhost string, lport uint32, rport uint32, hash string, debug bool) (uint32, error) {

	// refer to https://godoc.org/golang.org/x/crypto/ssh for other authentication types
	sshConfig := &ssh.ClientConfig{
		User: u,
		Auth: []ssh.AuthMethod{
        ssh.Password(pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

    // remote SSH server
    var serverEndpoint = Endpoint{
        Host: rhost,
        Port: 22,
    }
    
	// Listen on remote server port
    //instance, _ := strconv.Atoi(os.Getenv("INSTANCE"))
    bindAddr := os.Getenv("BINDADDR")
    
    // remote forwarding port (on remote SSH server network)
    var serviceEndpoint = Endpoint{
        Host: bindAddr,
        Port: rport,
    }

//    quit := make(chan bool)
    
    // Connect to SSH remote server using serverEndpoint    
    conn, err := ssh.Dial("tcp", serverEndpoint.String(), sshConfig)
	if err != nil {
		log.Println(fmt.Printf("Dial INTO remote server error: %s", err))
        return rport, err
	}

    /*
    go func() {
        conn.Wait()
        log.Println("Local Client: SSH COnnection closed.")
        quit <- true
    }()
    */
    
    // Listen on remote server port
	listener, err := conn.Listen("tcp", serviceEndpoint.String())
	if err != nil {
		log.Println(fmt.Printf("Listen open port ON remote server error: %s", err))
        return rport, err
	}

    _, rListenerPort, _ := utils.GetHostPort(listener.Addr())
    
    // Store channel in connection store for easy retival.
    clients[hash] = listener

    
    go func(){
        
        // handle incoming connections on reverse forwarded tunnel
        for {
            remote, err := listener.Accept()
            if err != nil {
                log.Println(err)
                return
            }

            go func(remote net.Conn) {
                fmt.Println("Local Server: New Connection from", remote.RemoteAddr())
                rhost, _, err := utils.GetHostPort(remote.RemoteAddr()) 
                if err != nil {
                    log.Println(err)
                    return
                }

                ip := net.ParseIP(rhost)

                tcpAddr := &net.TCPAddr {
                    IP: ip,
                }   

                d := net.Dialer{LocalAddr: tcpAddr}

                local, err := d.Dial("tcp", serviceEndpoint.String())
                if err != nil {
                    log.Println(fmt.Printf("Dial INTO local service error: %s", err))
                    remote.Close()
                    return
                }		

                handleClient(remote, local)
                
            }(remote)
        }        
    }()

    return uint32(rListenerPort), err
}


/*
 * Check if client is already connected
 */
func IsConnected(hash string) bool {
    _, ok := clients[hash]
    
    if ok {
        return true
    }
    
    return false
}

/*
 * Diconnect client
 */
func Disconnect(hash string) {
    ch := clients[hash]

    if ch != nil {
        err := ch.Close()
        utils.Check(err) 
        delete(clients, hash)
        clients[hash] = nil
    }
}
