package connmonitor

import (
    "fmt"
    "log"
    "regexp"
    "encoding/json"
    
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
}

var connections map[int32]*Host

type ConnAddedEvent   func(p int32, h Host)
type ConnRemovedEvent func(p int32)
 
func AddConnection(ev * psnotify.ProcEventFork, a ConnAddedEvent) {
    ppid := int32(ev.ParentPid)
    cpid := int32(ev.ChildPid)
    var pid int32
    
    proc, _ := ps.NewProcess(ppid)
    cmdline, _ := proc.CmdlineSlice()

    if len(cmdline) > 0 {
        matched, _ := regexp.MatchString("sshd: ci@notty*", cmdline[0])
        if matched {
            fmt.Println(cmdline)
            conns, _ := netutil.ConnectionsPid("inet", ppid)

            //Declare host to store connection information
            var host Host
            
            for conn := range conns {
                if conns[conn].Family == 2 && conns[conn].Status == "LISTEN" {
                    
                    pid = conns[conn].Pid
                    host.LocalPort = int32(conns[conn].Laddr.Port)
                    log.Println(host)
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
            
            //Bug: Race Condition. The EXEC event can be missed
            //Need to find out way to deterministicaly make sure 
            //child process has executed and new command line is
            //available
            execev := <-watcher.Exec    
            if (int32(execev.Pid) == cpid) {
                childproc, _ := ps.NewProcess(cpid)
                childcmdline, _ := childproc.CmdlineSlice()
                configstr := childcmdline[len(childcmdline) - 1]
                config := Config{}
                json.Unmarshal([]byte(configstr), &config)
                fmt.Println(config)
                host.RemotePort = config.Port
                host.Config = config
            }

            fmt.Printf("Host %s:%d Connected on Port %d\n", 
                       host.RemoteIP, 
                       host.RemotePort, 
                       host.LocalPort)

            //Send AddedEvent Callback.
            connections[pid] = &host
            a(pid, host)
            
        }
    }
}

func RemoveConnection(ev * psnotify.ProcEventExit, r ConnRemovedEvent) {
    pid := int32(ev.Pid)
    
    _, ok := connections[pid]
    if ok {
        delete(connections, pid)
        //Send RemoveEvent Callback
        r(pid)        
    }
}

func handleEvents(pid int, a ConnAddedEvent, r ConnRemovedEvent) {

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
                AddConnection(ev, a)
            case <-watcher.Exec:
            case ev := <-watcher.Exit:
                RemoveConnection(ev, r)

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

func Monitor(a ConnAddedEvent, r ConnRemovedEvent) {
    
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
                handleEvents(int(pids[pid]), a, r)
            }            
        }
    }
}