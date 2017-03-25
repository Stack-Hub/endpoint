package connmonitor

import (
    "fmt"
    "regexp"
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
    LocalPort  int32
    RemoteIP   string
    RemotePort int32
    Config     Config
    cpid       int
}

var connections map[int32]*Host

type ConnAddedEvent   func(p int32, h Host)
type ConnRemovedEvent func(p int32, h Host)
 
func AddConnection(ev * psnotify.ProcEventFork, username string, a ConnAddedEvent) {
    ppid := int32(ev.ParentPid)
    cpid := int32(ev.ChildPid)
    var pid int32
    
    proc, _ := ps.NewProcess(ppid)
    cmdline, _ := proc.CmdlineSlice()

    if len(cmdline) > 0 {
        procName := "sshd: " + username + "@notty"
        matched, _ := regexp.MatchString(procName, cmdline[0])
        if matched {
            conns, _ := netutil.ConnectionsPid("inet", ppid)
            fmt.Println(conns)

            //Declare host to store connection information
            var host Host
            
            for conn := range conns {
                if conns[conn].Family == 2 && conns[conn].Status == "LISTEN" {
                    
                    pid = conns[conn].Pid
                    host.LocalPort = int32(conns[conn].Laddr.Port)
                }
                
                if conns[conn].Family == 2 && conns[conn].Status == "ESTABLISHED" {
                    host.RemoteIP = conns[conn].Raddr.IP
                }
                
            }
            
            watcher, err := psnotify.NewWatcher()
            if err != nil {
                fmt.Println(err)
            }

            err = watcher.Watch(ev.ChildPid, psnotify.PROC_EVENT_EXEC)
            if err != nil {
                fmt.Println(err)
            }

            for ;; {
                childproc, _ := ps.NewProcess(cpid)
                childcmdline, _ := childproc.Cmdline()
                fmt.Println(childcmdline)

                regex := regexp.MustCompile(`^(tail -F ({.*}))$`)
                m := regex.FindStringSubmatch(childcmdline)
                fmt.Println(m)
                if len(m) == 0 {
                    <-watcher.Exec
                    continue
                }
                
                childcmdlineArray, _ := childproc.CmdlineSlice()
                configstr := childcmdlineArray[len(childcmdlineArray) - 1]
                fmt.Println(configstr)
                config := Config{}
                json.Unmarshal([]byte(configstr), &config)
                host.RemotePort = config.Port                
                host.Config = config
                host.cpid = int(cpid)
                break
            }

            //Send AddedEvent Callback.
            connections[pid] = &host
            a(pid, host)
            
        }
    }
}

func RemoveConnection(ev * psnotify.ProcEventExit, r ConnRemovedEvent) {
    pid := int32(ev.Pid)
    
    h, ok := connections[pid]
    if ok {
        proc, err := os.FindProcess(h.cpid)
        if err != nil {
            fmt.Println(err)
        }
        
        proc.Kill()
        
        delete(connections, pid)
        //Send RemoveEvent Callback
        r(pid, *h)
    }
}

func handleEvents(username string, pid int, a ConnAddedEvent, r ConnRemovedEvent) {

    // New Process Watcher. 
    watcher, err := psnotify.NewWatcher()
    if err != nil {
        fmt.Println(err)
    }

    // Process fork, exec, exit & error events
    go func() {
        for {
            select {
            case ev := <-watcher.Fork:
                go AddConnection(ev, username, a)
            case <-watcher.Exec:
            case ev := <-watcher.Exit:
                go RemoveConnection(ev, r)

            case <-watcher.Error:
            }
        }
    }()

    // Process fork, exec, exit & error events
    err = watcher.Watch(pid, psnotify.PROC_EVENT_ALL)
    if err != nil {
        fmt.Println(err)
    }
}

func Monitor(username string, a ConnAddedEvent, r ConnRemovedEvent) {
    
    connections = make(map[int32]*Host)
    
    // Get the list of all Pids in the system 
    // and search for sshd process.
    pids, _ := ps.Pids()
    for pid := range pids  {
        proc, _ := ps.NewProcess(pids[pid])
        cmdline, _ := proc.CmdlineSlice()

        if len(cmdline) > 0 {
            // For each sshd process Handle fork, exec & exit 
            // events for all child processes.
            matched, _ := regexp.MatchString(`/usr/sbin/sshd`, cmdline[0])
            if matched {    
                fmt.Printf("Monitoring %d for Connections\n", pids[pid])
                handleEvents(username, int(pids[pid]), a, r)
            }            
        }
    }
}