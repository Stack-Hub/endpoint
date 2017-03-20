package main

import (
    "fmt"
    "regexp"
    
    "github.com/mindreframer/golang-stuff/github.com/jondot/gosigar/psnotify"
    netutil "github.com/shirou/gopsutil/net"
    ps "github.com/shirou/gopsutil/process"

)

func connections(ev * psnotify.ProcEventFork) {
    ppid := int32(ev.ParentPid)
    cpid := int32(ev.ChildPid)
    proc, _ := ps.NewProcess(ppid)
    cmdline, _ := proc.CmdlineSlice()

    if len(cmdline) > 0 {
        matched, _ := regexp.MatchString("sshd: ci@notty*", cmdline[0])
        if matched {
            fmt.Println(cmdline)
            conns, _ := netutil.ConnectionsPid("inet", ppid)
            for conn := range conns {
                fmt.Println(conns[conn])
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
                fmt.Println(childcmdline)                            
            }
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
                connections(ev)
            case <-watcher.Exec:
            case <-watcher.Exit:
            case <-watcher.Error:
            }
        }
    }()

    err = watcher.Watch(pid, psnotify.PROC_EVENT_ALL)
    if err != nil {
        fmt.Println(err)
    }
}

func main() {

    pids, _ := ps.Pids()
    for pid := range pids  {
        fmt.Printf("Checking %d", pid)
        proc, _ := ps.NewProcess(int32(pid))
        cmdline, _ := proc.CmdlineSlice()
        matched, _ := regexp.MatchString("*/sshd", cmdline[0])
        if matched {    
            fmt.Printf("Watching %d", pid)
            watchSSH(pid)
        }
    }
    
    var num int
    /* ... do stuff ... */
    fmt.Scanf("%d", &num)
}