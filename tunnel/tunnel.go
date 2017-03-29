package tunnel

import (
    "fmt"
    "encoding/json"
    "os"
    "strconv"
    
    "github.com/mindreframer/golang-stuff/github.com/jondot/gosigar/psnotify"
    netutil "github.com/shirou/gopsutil/net"
    ps "github.com/shirou/gopsutil/process"
)

type Config struct {
    Port int32  `json:"port"`
}

type Host struct {
    LocalPort  int32  `json:"lport"`
    RemoteIP   string `json:"raddr"`
    RemotePort int32  `json:"rport"`
    Config     Config `json:"config"`
    Uid        int    `json:"uid"`
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
    pid  := os.Getpid()
    
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
    host.Uid = os.Getuid()
    
    fmt.Println(host)
    
    namedPipe := "/tmp/" + strconv.Itoa(pid)
    stdout, _ := os.OpenFile(namedPipe, os.O_RDWR, 0600)
    
    payload, _ := json.Marshal(host)
    
    stdout.Write(payload)
    stdout.Close()
    waitForPPidExit(int(ppid))

}
