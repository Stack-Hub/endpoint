/* Copyright (C) Ashish Thakwani - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Ashish Thakwani <athakwani@gmail.com>, October 2017
 */
package server

import (
	"fmt"
    "net"
    "log"
    "os"
    "crypto/rsa" 
    "crypto/rand" 
    "encoding/pem" 
    "crypto/x509" 

	"golang.org/x/crypto/ssh"
    "github.com/duppercloud/trafficrouter/omap"
    "github.com/duppercloud/trafficrouter/utils"
//    log "github.com/Sirupsen/logrus"
)

type Callback func(*omap.OMap, *utils.Host)

const (
	RemoteForwardRequest       = "tcpip-forward"        
	ForwardedTCPReturnRequest  = "forwarded-tcpip"      
	CancelRemoteForwardRequest = "cancel-tcpip-forward" 
)

/*
 * User record
 */
type user struct {
    user string
    ccb Callback
    dcb Callback
    m   *omap.OMap
}

/*
 * User DB
 */
var userDB map[string]user

/*
 * tcpipForward is structure for RFC 4254 7.1 "tcpip-forward" request
 */
type tcpipForward struct {
	Host   string
	Port   uint32
    Instance uint32 
    Label    string 
}

/*
 * tcpipForwardResponse is structure for RFC 4254 7.1 "tcpip-forward" response
 */
type tcpipForwardResponse struct {
    Port uint32
    Host string
}

/* 
 * directForward is struxture for RFC 4254 7.2 - can be used for "forwarded-tcpip" and "direct-tcpip" 
 */
type directForward struct {
	Host1 string
	Port1 uint32
	Host2 string
	Port2 uint32
}

/* 
 * SSH Connection to external clients
 */
type RemoteForward struct {
    Metadata tcpipForward
    channel ssh.Channel
}

/*
 * MakeSSHKeyPair make a pair of public and private keys for SSH access.
 * Public key is encoded in the format for inclusion in an OpenSSH authorized_keys file.
 * Private Key generated is PEM encoded
 */
func MakeSSHKeyPair() ([]byte, []byte, error) {
    privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
    if err != nil {
        return nil, nil, err
    }

    privateKeyBlock := pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
    privateKeyPEM := pem.EncodeToMemory(&privateKeyBlock)
    
    // generate and write public key
    pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
    if err != nil {
        return nil, nil, err
    }
    
    return privateKeyPEM, ssh.MarshalAuthorizedKey(pub), nil
}

/*
 * Reject all channel Request with Prohibited error
 */
func ProhibitChannels(chans <-chan ssh.NewChannel) {
    for ch := range chans {
            ch.Reject(ssh.Prohibited, "")
    }
}

/*
 * Open reverse channel on SSH and forward data.
 */
func ReverseChannel(conn net.Conn, sshConn *ssh.ServerConn) error { 
    p := directForward{}
    var err error

    var portnum int
    p.Host1, portnum, err = utils.GetHostPort(conn.LocalAddr())
    if err != nil {
        return err
    }    
    p.Port1 = uint32(portnum)

    p.Host2, portnum, err = utils.GetHostPort(conn.RemoteAddr())
    if err != nil {
        return err
    }

    p.Port2 = uint32(portnum)

    fmt.Println(p)
    ch, reqs, err := sshConn.OpenChannel(ForwardedTCPReturnRequest, ssh.Marshal(p))
    if err != nil {
        log.Println("Open forwarded Channel: ", err.Error())
        return err
    }
    go ssh.DiscardRequests(reqs)

    close := func() {
        ch.Close()
        conn.Close()
    }

    utils.CopyReadWriters(conn, ch, close)
    return nil
}

/*
 * Handle incoming connections on this new listener
 */
func ForwardConnections(ln net.Listener, sshConn *ssh.ServerConn, done chan bool) {     
    for {
        conn, err := ln.Accept()
        if err != nil { // Unable to accept new connection - listener likely closed
            fmt.Println("Listener Closed asynchronously", err)
            return
        }

        go ReverseChannel(conn, sshConn)
    }
}


/*
 * Handle incoming Remote Forward requests
 */
func HandleRemoteForwardRequest(req *ssh.Request, sshConn *ssh.ServerConn, done chan bool, u *user) error {
    t := tcpipForward{}

    ssh.Unmarshal(req.Payload, &t)
    addr := fmt.Sprintf("%s:%d", t.Host, t.Port)
    ln, err := net.Listen("tcp", addr) //tie to the client connection

    if err != nil {
        log.Println("Unable to listen on address: ", addr)
        return err
    }
    
    log.Println("Listening on address: ", ln.Addr().String())

    _, port, err := utils.GetHostPort(ln.Addr())
    if err != nil {
        return err
    }

    // Send reply for 'tcpip-forward' request
    res := tcpipForwardResponse{}
    res.Port = uint32(port)
    res.Host = os.Getenv("BINDADDR")
    req.Reply(true, ssh.Marshal(res))

    h := &utils.Host{}
    h.ListenPort = res.Port
    h.RemoteIP = t.Host
    h.RemotePort = t.Port
    h.Config.Label = t.Label
    h.Config.Instance = t.Instance
    h.Config.Port = t.Port
    
    u.ccb(u.m, h)
    
    go func() {
       select {
        case <-done:
            u.dcb(u.m, h)
           ln.Close()
        } 
    }()
    
    ForwardConnections(ln, sshConn, done)

    return nil 
}

/*
 * Handle incoming requests
 */
func HandleRequets(in <-chan *ssh.Request, sshConn *ssh.ServerConn, done chan bool, u *user) {        
    for req := range in {
        switch req.Type {
            case RemoteForwardRequest:
                go HandleRemoteForwardRequest(req, sshConn, done, u)
            case CancelRemoteForwardRequest:
                done <- true
                req.Reply(true, nil)
            default:
                req.Reply(false, nil)
        }
    }    
}

func HandleClient(nConn net.Conn, config *ssh.ServerConfig) {
    // Before use, a handshake must be performed on the incoming
    // net.Conn.
    sshConn, chans, reqs, err := ssh.NewServerConn(nConn, config)
    if err != nil {
        log.Fatal("failed to handshake: ", err)
    }

    // Send Reject with Prohibited error to all channels.
    go ProhibitChannels(chans)

    user := userDB[sshConn.Permissions.Extensions["user"]]
    done := make(chan bool)

    // Handle Requests
    go HandleRequets(reqs, sshConn, done, &user)

    sshConn.Wait()
    done <- true
    fmt.Println("Stop forwarding/listening")                    
}

func Listen() (error) {

    // Handle Authentication
    config := &ssh.ServerConfig{
        PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
            if u, ok := userDB[c.User()]; ok && string(pass) == "123456789" {
                return &ssh.Permissions{Extensions: map[string]string{"user": u.user}}, nil
            }
            return nil, fmt.Errorf("password rejected for %q", c.User())
        },
    }

    priv, _, err := MakeSSHKeyPair()

    private, err := ssh.ParsePrivateKey(priv)
    if err != nil {
        log.Fatal("Failed to parse private key: ", err)
    }

    config.AddHostKey(private)
        
    // Listen & Accept connections
    listener, err := net.Listen("tcp", "0.0.0.0:22")
    if err != nil {
        log.Fatal("failed to listen for connection: ", err)
    }

    for {
        nConn, err := listener.Accept()
        if err != nil {
            log.Fatal("failed to accept incoming connection: ", err)
        }

        go HandleClient(nConn, config)
    }
    
    return nil
}

func AddUser(uname string, m *omap.OMap, ccb Callback, dcb Callback) {
    u := user{}
    u.user = uname
    u.m = m
    u.ccb = ccb
    u.dcb = dcb
    
    // Add user to database
    userDB[uname] = u
}