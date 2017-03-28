package main

import (
    "fmt"
    "encoding/json"
    "os"
    
    "github.com/mindreframer/golang-stuff/github.com/jondot/gosigar/psnotify"
    netutil "github.com/shirou/gopsutil/net"
    ps "github.com/shirou/gopsutil/process"
)

type Config struct {
    Port int32  `json:"port"`
    Mode string `json:"mode"`
}

type Host struct {
    Username   string
    LocalPort  int32
    RemoteIP   string
    RemotePort int32
    Config     Config
    PPid        int
}

func waitForPPidExit(ppid int) {
    watcher, err := psnotify.NewWatcher()
    if err != nil {
        fmt.Println(err)
    }

    err = watcher.Watch(ppid, psnotify.PROC_EVENT_EXIT)
    if err != nil {
        fmt.Println(err)
    }

    for ;; {
        ev := <-watcher.Exit
        if (ev.Pid == ppid) {
            os.Exit(0)            
        }
    }

    
}

func SendConfig() {
    ppid := int32(os.Getppid())
    fmt.Println("ppid = ", ppid)
    
    pproc, _ := ps.NewProcess(ppid)
    pcmdline, _ := pproc.CmdlineSlice()
    fmt.Println("Parent Process cmdline = ", pcmdline)
    
    conns, _ := netutil.ConnectionsPid("inet", ppid)

    //Declare host to store connection information
    var host Host

    for conn := range conns {
        if conns[conn].Family == 2 && conns[conn].Status == "LISTEN" {
            host.LocalPort = int32(conns[conn].Laddr.Port)
        }

        if conns[conn].Family == 2 && conns[conn].Status == "ESTABLISHED" {
            host.RemoteIP = conns[conn].Raddr.IP
        }

    }

    configstr := os.Args[len(os.Args) - 1]
    fmt.Println(configstr)
    config := Config{}
    json.Unmarshal([]byte(configstr), &config)
    host.RemotePort = config.Port                
    host.Config = config
    host.PPid = int(ppid)
    
    fmt.Println(host)
    waitForPPidExit(int(ppid))

}

func main () {
    SendConfig()
}