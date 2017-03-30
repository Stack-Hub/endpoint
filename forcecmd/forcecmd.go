package forcecmd

import (
    "fmt"
    "encoding/json"
    "os"
    "regexp"
    "strconv"
    "net"
    "log"
    
    "github.com/mindreframer/golang-stuff/github.com/jondot/gosigar/psnotify"
    netutil "github.com/shirou/gopsutil/net"
    ps "github.com/shirou/gopsutil/process"
)

type Config struct {
    Port int32  `json:"port"`
}

type Host struct {
    ListenPort  int32  `json:"lisport"`
    RemoteIP    string `json:"raddr"`
    RemotePort  uint32  `json:"rport"`
    ServicePort int32  `json:"sport"`
    Config      Config `json:"config"`
    Uid         int    `json:"uid"`
    Pid         int32  `json:"pid"`
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
        fmt.Println("Parent Died ", ev)
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
    fmt.Println(conns)

    //Declare host to store connection information
    var host Host
    host.Pid = int32(pid)

    for conn := range conns {
        if conns[conn].Family == 2 && conns[conn].Status == "LISTEN" {
            host.ListenPort = int32(conns[conn].Laddr.Port)
        }

        if conns[conn].Family == 2 && conns[conn].Status == "ESTABLISHED" {
            host.RemoteIP = conns[conn].Raddr.IP
            host.RemotePort = conns[conn].Raddr.Port
        }

    }

    configstr := os.Args[len(os.Args) - 1]
    fmt.Println(configstr)
    config := Config{}
    json.Unmarshal([]byte(configstr), &config)
    host.ServicePort = config.Port                
    host.Config = config
    host.Uid = os.Getuid()
    
    fmt.Println(host)

    
    pids, _ := ps.Pids()
    for pid := range pids  {
        proc, _ := ps.NewProcess(pids[pid])
        cmdline, _ := proc.Cmdline()

        if len(cmdline) > 0 {
            // For each sshd process Handle fork, exec & exit 
            // events for all child processes.
            mstr := `trafficrouter -s -u .* -uid %d .*`
            mstr = fmt.Sprintf(mstr, os.Getuid())
            matched, _ := regexp.MatchString(mstr, cmdline)
            if matched {    
                fmt.Println(cmdline)
                fmt.Printf("Found Server Process %d\n", pids[pid])
                f := "/tmp/" + strconv.Itoa(int(pids[pid])) + ".sock"
                log.Println("Sending data to ", f)
                c, err := net.Dial("unix", f)

                if err != nil {
                    panic(err)
                }
                
                payload, _ := json.Marshal(host)
                _, err = c.Write(payload)

                if err != nil {
                    log.Println(err)
                }            
                
                c.Close()
            }            
        }
    }
    
    waitForPPidExit(int(ppid))

}
