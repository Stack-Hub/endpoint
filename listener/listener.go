/* Copyright (C) Ashish Thakwani - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Ashish Thakwani <athakwani@gmail.com>, August 2017
 */
package main

import (
    "C"
    "fmt"
    "os"
    "syscall"
    "strconv"

    "github.com/duppercloud/trafficrouter/portmap"
    "github.com/rainycape/dl"
)

// go build -buildmode=c-shared -o listener.so listener.go

// Initliaze port map to get events of new port mappings.
var pmap, _ = portmap.New(os.Getenv("REPO_NAME"), false, true)

func main() {}

//export listen
func listen(fd C.int, backlog C.int) int32 {
    
    lib, err := dl.Open("libc", 0)
    if err != nil {
        return 0
    }
    defer lib.Close()

    var reallisten func(fd C.int, backlog C.int) int32
    lib.Sym("listen", &reallisten)

    sock, err := syscall.Getsockname(int(fd))
    if (err != nil) {
        return reallisten(fd, backlog)        
    }
    
    switch sock.(type) {
        case *syscall.SockaddrInet4:
            pmap.Add(strconv.Itoa(sock.(*syscall.SockaddrInet4).Port), "0")

        /* Only v4 is supported for now.
        case *syscall.SockaddrInet6:
            pmap.Add(strconv.Itoa(sock.(*syscall.SockaddrInet6).Port), "0")
            fmt.Println("Litening detected on ", strconv.Itoa(sock.(*syscall.SockaddrInet6).Port))
        */
    }

    return reallisten(fd, backlog)
}

//export close
func close(fd C.int) int32 {
    
    lib, err := dl.Open("libc", 0)
    if err != nil {
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
            pmap.Delete(strconv.Itoa(sock.(*syscall.SockaddrInet4).Port))

        /* Only v4 is supported for now.
        case *syscall.SockaddrInet6:
            pmap.Add(strconv.Itoa(sock.(*syscall.SockaddrInet6).Port), "0")
            fmt.Println("Litening detected on ", strconv.Itoa(sock.(*syscall.SockaddrInet6).Port))
        */
    }

    return realclose(fd)
}
