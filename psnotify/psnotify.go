package main

import (
    "fmt"
    "log"
    "regexp"
    "container/list"
    "strconv"
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
    localPort  int32
    remoteIP   string
    remotePort int32
    config     Config
    element  * list.Element
}

type Tunnels struct {
    idx   * list.Element
    hosts map[int32]*Host
    list  * list.List
}


var tunnels Tunnels
 
func AddConnection(ev * psnotify.ProcEventFork) {
    ppid := int32(ev.ParentPid)
    cpid := int32(ev.ChildPid)
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
                    e := tunnels.list.PushBack(conns[conn].Pid)
                    
                    host.localPort = int32(conns[conn].Laddr.Port)
                    host.element = e
                    
                    tunnels.hosts[conns[conn].Pid] = &host                    
                    log.Println(host)
                }
                
                if conns[conn].Family == 2 && conns[conn].Status == "ESTABLISHED" {
                    host.remoteIP = conns[conn].Raddr.IP
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

            execev := <-watcher.Exec    
            if (int32(execev.Pid) == cpid) {
                childproc, _ := ps.NewProcess(cpid)
                childcmdline, _ := childproc.CmdlineSlice()
                configstr := childcmdline[len(childcmdline) - 1]
                config := Config{}
                json.Unmarshal([]byte(configstr), &config)
                fmt.Println(config)
                host.remotePort = config.Port
            }

            fmt.Printf("Host %s:%d Connected on Port %d\n", 
                       host.remoteIP, 
                       host.remotePort, 
                       host.localPort)
            
        }
    }
}

func RemoveConnection(ev * psnotify.ProcEventExit) {
    pid := int32(ev.Pid)
    host, ok := tunnels.hosts[pid]
    if ok {
        log.Println("Deleting", tunnels.hosts[pid])

        //If tunnel index points to current index, then move the index forward.
        if tunnels.idx == host.element {
            tunnels.idx = tunnels.idx.Next()
        }   
    
        //Remove index from the list
        tunnels.list.Remove(host.element)

        //Delete host entry from map
        delete(tunnels.hosts, pid)

        fmt.Println(tunnels)
        pidlist := ""
        for e := tunnels.list.Front(); e != nil; e = e.Next() {
            pidlist += strconv.Itoa(int(e.Value.(int32))) + " "
        }        
        if len(pidlist) > 0 {
            log.Println("list>", pidlist)
        }
    }
}

func watchSSH(pid int) {
    watcher, err := psnotify.NewWatcher()
    if err != nil {
        fmt.Println(err)
    }

    // Process events
    go func() {
        for {
            select {
            case ev := <-watcher.Fork:
                AddConnection(ev)
            case <-watcher.Exec:
            case ev := <-watcher.Exit:
                RemoveConnection(ev)

            case <-watcher.Error:
            }
        }
    }()

    err = watcher.Watch(pid, psnotify.PROC_EVENT_ALL)
    if err != nil {
        fmt.Println(err)
    }
}

func Next() *Host {
    if len(tunnels.hosts) == 0 {
        return nil
    }
    
    if tunnels.idx == nil {
        tunnels.idx = tunnels.list.Front()
    }
        
    host := tunnels.hosts[tunnels.idx.Value.(int32)]
    
    if tunnels.idx.Next() == nil {
        tunnels.idx = tunnels.list.Front()
    } else {
        tunnels.idx = tunnels.idx.Next()
    }
    
    return host
}

func main() {

    tunnels.hosts = make(map[int32]*Host, 1)
    tunnels.idx = nil
    tunnels.list = list.New()
    tunnels.list.Init()

    pids, _ := ps.Pids()
    for pid := range pids  {
        proc, _ := ps.NewProcess(pids[pid])
        cmdline, _ := proc.CmdlineSlice()
        if len(cmdline) > 0 {
            matched, _ := regexp.MatchString(`/usr/sbin/sshd`, cmdline[0])
            if matched {    
                fmt.Printf("Watching %d\n", pids[pid])
                watchSSH(int(pids[pid]))
            }            
        }
    }
    
    num := 0
    
    for num != 10 {
        /* ... do stuff ... */
        fmt.Scanf("%d", &num)
        
        if num == 1 {
            fmt.Println(Next())
        }
        
    }
}