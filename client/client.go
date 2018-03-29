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
    "github.com/pipecloud/endpoint/utils"
    "github.com/prometheus/common/log"
)


type Connection struct {
    l net.Listener
    c *ssh.Client
}

/*
 * Connection store
 */
var clients map[string]*Connection = make(map[string]*Connection, 1)

func handleClient(client net.Conn, remote net.Conn) {
	defer client.Close()
	chDone := make(chan bool)

	// Start remote -> local data transfer
	go func() {
		_, err := io.Copy(client, remote)
		if err != nil {
			log.Debug(fmt.Sprintf("error while copy remote->local: %s", err))
		}
		chDone <- true
	}()

	// Start local -> remote data transfer
	go func() {
		_, err := io.Copy(remote, client)
		if err != nil {
			log.Debug(fmt.Sprintf("error while copy local->remote: %s", err))
		}
		chDone <- true
	}()

	<-chDone
}

func Connect(u string, pass string, rhost string, lport uint32, rport uint32, hash string, debug bool) error {

	sshConfig := &ssh.ClientConfig{
		User: u,
		Auth: []ssh.AuthMethod{
        ssh.Password(pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

    // remote SSH server
    var serverEndpoint = utils.Endpoint{
        Host: rhost,
        Port: 22,
    }
    
	// Listen on remote server port
    bindAddr := os.Getenv("BINDADDR")
    
    // remote forwarding port (on remote SSH server)
    var serviceEndpoint = utils.Endpoint{
        Host: bindAddr,
        Port: rport,
    }

    
    fmt.Println("SSH Client: Initiating connection to ", serverEndpoint.String())
    // Connect to SSH remote server using serverEndpoint    
    conn, err := ssh.Dial("tcp", serverEndpoint.String(), sshConfig)
	if err != nil {
		log.Debug(fmt.Printf("Dial INTO remote server error: %s", err))
        return err
	}
    
    // Listen on remote server port
    listener, err := conn.Listen("tcp", serviceEndpoint.String())
    if err != nil {
        log.Debug(fmt.Printf("Listen open port ON remote server error: %s", err))
        return err
    }

    fmt.Println("SSH Client: Listening connection on ", serverEndpoint.String(), 
                "@", serverEndpoint.String())
    // Store channel in connection store for easy retival.
    
    clients[hash] = &Connection{l: listener, c: conn}


    go func(){

        // handle incoming connections on reverse forwarded tunnel
        for {
            remote, err := listener.Accept()
            if err != nil {
                log.Debug("SSH Client: Remote Listener closed on ", serverEndpoint.String(), " with ", err)
                delete(clients, hash)
                log.Debug("SSH Client: clients ", clients)
                return
            }

            fmt.Println("SSH Client: Incoming connection on ", remote.LocalAddr().String(), 
                        " from ", listener.Addr().String())
            go func(remote net.Conn) {
                rhost, _, err := utils.GetHostPort(remote.RemoteAddr()) 
                if err != nil {
                    log.Debug(err)
                    return
                }

                ip := net.ParseIP(rhost)

                tcpAddr := &net.TCPAddr {
                    IP: ip,
                }   

                d := net.Dialer{LocalAddr: tcpAddr}

                // Service Port at remote container
                serviceEndpoint.Port = lport

                fmt.Println("SSH Client: Connecting to ", serviceEndpoint.String())
                local, err := d.Dial("tcp", serviceEndpoint.String())
                if err != nil {
                    log.Debug(fmt.Printf("Dial INTO local service error: %s", err))
                    remote.Close()
                    return
                }		

                fmt.Println("SSH Client: Routing Data between ", remote.LocalAddr().String(), 
                            " from ", local.RemoteAddr().String())
                handleClient(remote, local)
            }(remote)
        }        
    }()
        
    return nil
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
    connection := clients[hash]

    if connection != nil {
        fmt.Println("Request: Closing connections ", connection)
        connection.l.Close()
        connection.c.Close()
        delete(clients, hash)
    }
}
