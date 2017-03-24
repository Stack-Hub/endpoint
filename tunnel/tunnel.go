package main

import (
    "fmt"
    "os"
    "net"
    "strings"
    "strconv"
    
    netutil "github.com/shirou/gopsutil/net"
    ps "github.com/shirou/gopsutil/process"

)

//function to get the public ip address
func GetOutboundIP() string {
    conn, _ := net.Dial("udp", "8.8.8.8:80")
//    HandleError("net.Dial: ",err)
    defer conn.Close()
    localAddr := conn.LocalAddr().String()
    idx := strings.LastIndex(localAddr, ":")
    return localAddr[0:idx]
}

func main() {
    ip := GetOutboundIP()
    fmt.Println(ip)
    
    p, _ := strconv.Atoi(os.Args[1])
    pid := int32(p)
    fmt.Println(pid)
    
    proc, _ := ps.NewProcess(pid)
    fmt.Println(proc.Cmdline())
    
    conns, _ := netutil.ConnectionsPid("inet", pid)

    for conn := range conns {
        fmt.Println(conns[conn])
    }
}