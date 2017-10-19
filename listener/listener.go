/* Copyright (C) Ashish Thakwani - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Ashish Thakwani <athakwani@gmail.com>, August 2017
 */
package main

import (
    "C"
    "os"
    "syscall"
    "log"
    "net/rpc"
    
    "github.com/duppercloud/trafficrouter/opt/register"
    "github.com/rainycape/dl"
)

// go build -buildmode=c-shared -o listener.so listener.go

func main() {}

var client *rpc.Client

//export listen
func listen(fd C.int, backlog C.int) int32 {
    
    lib, err := dl.Open("libc", 0)
    if err != nil {
        log.Println("Error opening libc", err)
        return 0
    }
    defer lib.Close()

    var reallisten func(fd C.int, backlog C.int) int32
    lib.Sym("listen", &reallisten)

    sock, err := syscall.Getsockname(int(fd))
    if (err != nil) {
        log.Println("Error getting socket name", err)
        return reallisten(fd, backlog)        
    }
    
    switch sock.(type) {
        case *syscall.SockaddrInet4:
            var err error
            port := uint32(sock.(*syscall.SockaddrInet4).Port)
            if client == nil {
                client, err = rpc.DialHTTP("tcp", "localhost:3877")
                if err != nil {
                    log.Println("dialing:", err)
                    return 107
                }            
            }
            
            args := &register.Args{Lport: port,
                                   Rport: port,}
            int errno
            client.Call("RPC.Connect", args, &errno)
        
        /* Only v4 is supported for now.
        case *syscall.SockaddrInet6:
            pmap.Add(strconv.Itoa(sock.(*syscall.SockaddrInet6).Port), "0")
            log.Println("Litening detected on ", strconv.Itoa(sock.(*syscall.SockaddrInet6).Port))
        */
    }

    return reallisten(fd, backlog)
}

//export close
func close(fd C.int) int32 {

    f, err := os.OpenFile("/tmp/log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
    if err != nil {
        return 0
    }
    defer f.Close()

    log.SetOutput(f)
    
    lib, err := dl.Open("libc", 0)
    if err != nil {
        log.Println("Error opening libc", err)
        return 0
    }
    defer lib.Close()

    var realclose func(fd C.int) int32
    lib.Sym("close", &realclose)

    sock, err := syscall.Getsockname(int(fd))
    if (err != nil) {
        log.Println("Error getting socket name", err)
        return realclose(fd)        
    }
    
    switch sock.(type) {
        case *syscall.SockaddrInet4:
            var err error
            port := uint32(sock.(*syscall.SockaddrInet4).Port)
            if client == nil {
                client, err = rpc.DialHTTP("tcp", "localhost:3877")
                if err != nil {
                    log.Println("dialing:", err)
                    return 107
                }            
            }

            args := &register.Args{Lport: port,
                                   Rport: port,}
            int errno
            client.Call("RPC.Disconnect", args, &errno)

        /* Only v4 is supported for now.
        case *syscall.SockaddrInet6:
            pmap.Add(strconv.Itoa(sock.(*syscall.SockaddrInet6).Port), "0")
            log.Println("Litening detected on ", strconv.Itoa(sock.(*syscall.SockaddrInet6).Port))
        */
    }

    return realclose(fd)
}
