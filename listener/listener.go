/* Copyright (C) Ashish Thakwani - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Ashish Thakwani <athakwani@gmail.com>, August 2017
 */
package main

import (
    "C"
    "fmt"
    "syscall"
    "encoding/json"
    "io/ioutil"    
    "strconv"

    "github.com/rainycape/dl"
)

// go build -buildmode=c-shared -o listener.so listener.go

type ports struct {
    m map[string]int `json:"list"`
}

var p ports

func main() {}

func load() *ports {
    
    var p ports
    
    file, e := ioutil.ReadFile("/tmp/ports")
    if e != nil {
        fmt.Printf("File error: %v\n", e)
        return &p
    }

    json.Unmarshal(file, &p)
    
    if len(p.m) == 0 {
        p.m = make(map[string]int, 1)
    }
    
    return &p
}

func save(p *ports) {
    
    jsonp, err := json.Marshal(p.m)
    if err != nil {
        fmt.Println(err)
    }
    
    fmt.Println(p)
    fmt.Println(jsonp)

    err = ioutil.WriteFile("/tmp/ports", jsonp, 0644)
    if err != nil {
        fmt.Println("listener.so: Error writing file")
    }
}

//export listen
func listen(fd C.int, backlog C.int) int32 {

    p := load()
    
    lib, err := dl.Open("libc", 0)
    if err != nil {
        return 0
    }
    defer lib.Close()

    var reallisten func(fd C.int, backlog C.int) int32
    lib.Sym("listen", &reallisten)

    sock, err := syscall.Getsockname(int(fd))
    if (err != nil) {
        fmt.Println(err)
        return reallisten(fd, backlog)        
    }
    
    switch sock.(type) {
        case *syscall.SockaddrInet4:
            f := strconv.Itoa(int(fd))
            p.m[f] = sock.(*syscall.SockaddrInet4).Port
        case *syscall.SockaddrInet6:
            f := strconv.Itoa(int(fd))
            p.m[f] = sock.(*syscall.SockaddrInet6).Port
    }
        
    save(p)

    return reallisten(fd, backlog)
}
