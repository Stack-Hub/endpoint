package main

import (
    "C"
    "syscall"
    "log"
    "net/rpc"
    "os"
    
    "github.com/microstacks/stack/endpoint/opt/register"
    "github.com/rainycape/dl"
)

// go build -buildmode=c-shared -o listener.so listener.go

func main() {}

var client *rpc.Client

var ports map[uint32]int = make(map[uint32]int, 1)

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
            var errno int
            client.Call("RPC.Connect", args, &errno)
            pid := os.Getpid()
            ports[port] = pid
            log.Println("Listening port opened by pid=", pid)
    }

    return reallisten(fd, backlog)
}

//export close
func close(fd C.int) int32 {
    
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
        return realclose(fd)
    }
    
    switch sock.(type) {
        case *syscall.SockaddrInet4:
            var err error
            port := uint32(sock.(*syscall.SockaddrInet4).Port)
        
            if _, ok := ports[port]; ok {
                if client == nil {
                    client, err = rpc.DialHTTP("tcp", "localhost:3877")
                    if err != nil {
                        log.Println("dialing:", err)
                        return 107
                    }
                }

                pid := os.Getpid()
                if ports[port] == pid {
                    log.Println("Listening port closed by pid=", pid)
                    args := &register.Args{Lport: port,
                                           Rport: port,}
                    var errno int
                    client.Call("RPC.Disconnect", args, &errno)
                    delete(ports, port)
                }
            }
    }

    return realclose(fd)
}
