package forcecmd

import (
    "fmt"
    "encoding/json"
    "os"
    "regexp"
    "strconv"
    "net"
    "log"
    "runtime"

    "../config"
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

func blockForever() {
    for {
        runtime.Gosched()
    }
}

func SendConfig() {
    ppid := int32(os.Getppid())
    fmt.Println("ppid = ", ppid)
    
    pproc, _ := ps.NewProcess(ppid)

    //Get SSH Proc PID
    spid, _ := pproc.Ppid()
    sproc, _ := ps.NewProcess(spid)
    scmdline, _ := sproc.CmdlineSlice()
    fmt.Println("Parent Process cmdline = ", scmdline)
    
    conns, _ := netutil.ConnectionsPid("inet", spid)
    fmt.Println(conns)

    //Declare host to store connection information
    var host Host
    host.Pid = int32(ppid)

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
    cfg := Config{}
    json.Unmarshal([]byte(configstr), &cfg)
    host.ServicePort = cfg.Port                
    host.Config = cfg
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
                f := config.RUNPATH + strconv.Itoa(int(pids[pid])) + ".sock"
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
    
    blockForever()

}
