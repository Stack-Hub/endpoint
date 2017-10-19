/* Copyright (C) Ashish Thakwani - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Ashish Thakwani <athakwani@gmail.com>, October 2017
 */
package client

import (
	"fmt"
	"io"
	"log"
	"net"
    "os"
    "errors"
    "strconv"

	"golang.org/x/crypto/ssh"
    "github.com/duppercloud/trafficrouter/utils"
)


/*
 * Connection store
 */
var clients map[string]ssh.Channel = make(map[string]ssh.Channel, 1)


type Endpoint struct {
	Host string
	Port uint32
}

func (endpoint *Endpoint) String() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

func handleClient(client ssh.Channel, remote net.Conn) {
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
    instance, _ := strconv.Atoi(os.Getenv("INSTANCE"))
    bindAddr := os.Getenv("BINDADDR")
    
    // remote forwarding port (on remote SSH server network)
    var serviceEndpoint = Endpoint{
        Host: bindAddr,
        Port: lport,
    }

    // Connect to SSH remote server using serverEndpoint    
    conn, err := net.DialTimeout("tcp", serverEndpoint.String(), sshConfig.Timeout)
	if err != nil {
		log.Println(fmt.Printf("Dial INTO remote server error: %s", err))
        return rport, err
	}
    
    c, chans, requests, err := ssh.NewClientConn(conn, serverEndpoint.String(), sshConfig)
	if err != nil {
		log.Println(fmt.Printf("Dial INTO remote server error: %s", err))
        return rport, err
	}
    
    go ssh.DiscardRequests(requests)

    // Once a Session is created, you can execute a single command on
    // the remote side using the Run method.
    type channelForwardMsg struct {
        IP  string
        Port uint32
        Instance uint32 
        Label    string 
    }
    
    
	m := channelForwardMsg{
        IP: serviceEndpoint.Host,
        Port: serviceEndpoint.Port,
        Instance: uint32(instance),
        Label: os.Getenv("LABEL"),
	}
    
    fmt.Println("tcpip-forward=", m)
    
    ok, resp, err := c.SendRequest("tcpip-forward", true, ssh.Marshal(&m))
	if err != nil {
        log.Println(err)
        return rport, err
	}
    
    if !ok {
        log.Println(errors.New("ssh: tcpip-forward request denied by peer"))
        return rport, err
	}

    type tcpipForwardResponse struct {
        Port uint32
        Host string
    }
    
	// If the original port was 0, then the remote side will
	// supply a real port number in the response.
    var bindResp tcpipForwardResponse
    ssh.Unmarshal(resp, &bindResp)
    fmt.Println("rcvd=", bindResp)
        
    for ch := range chans {
		var (
			err   error
		)    
        switch channelType := ch.ChannelType(); channelType {
            case "forwarded-tcpip":
                type forwardedTCPPayload struct {
                    Addr       string
                    Port       uint32
                    OriginAddr string
                    OriginPort uint32
                }

                var payload forwardedTCPPayload
                if err = ssh.Unmarshal(ch.ExtraData(), &payload); err != nil {
                    ch.Reject(2, "could not parse forwarded-tcpip payload: "+err.Error())
                    continue
                }

/*
                laddr, err = utils.ParseTCPAddr(payload.Addr, payload.Port)
                if err != nil {
                    ch.Reject(2, err.Error())
                    continue
                }
                raddr, err = utils.ParseTCPAddr(payload.OriginAddr, payload.OriginPort)
                if err != nil {
                    ch.Reject(2, err.Error())
                    continue
                }
*/
            
                channel, reqs, err := ch.Accept()
            
                // Store channel in connection store for easy retival.
                clients[hash] = channel

                go ssh.DiscardRequests(reqs)
            
                ip := net.ParseIP(bindResp.Host)

                tcpAddr := &net.TCPAddr{
                    IP: ip,
                }   

                d := net.Dialer{LocalAddr: tcpAddr}

                local, err := d.Dial("tcp", serviceEndpoint.String())
                if err != nil {
                    log.Fatalln(fmt.Printf("Dial INTO local service error: %s", err))
                }		
            
                go handleClient(channel, local)
        }
    }

    return bindResp.Port, err
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
    ch := clients[hash]

    if ch != nil {
        err := ch.Close()
        utils.Check(err) 
        delete(clients, hash)
        clients[hash] = nil
    }
}
