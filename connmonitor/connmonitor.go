package connmonitor

import (
    "fmt"
    "regexp"
    "encoding/json"
    "os"
    "strconv"
    "bytes"
    "io"
    "syscall"
    
    "github.com/mindreframer/golang-stuff/github.com/jondot/gosigar/psnotify"
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

var connections map[int32]*Host
var processing map[int32]bool


type ConnAddedEvent   func(p int32, h Host)
type ConnRemovedEvent func(p int32, h Host)
 
func AddConnection(ev * psnotify.ProcEventExec, uid int, a ConnAddedEvent) {
    pid := int32(ev.Pid)
    
    _, ok := processing[pid]
    if !ok {
        processing[pid] = true
    
        //Declare host to store connection information
        var host Host

        tunnelProc, _ := ps.NewProcess(pid)
        tunnelCmdline, _ := tunnelProc.CmdlineSlice()

        if len(tunnelCmdline) > 0 {
            procName := os.Args[0] + " -t"
            matched, _ := regexp.MatchString(procName, tunnelCmdline[0])
            if matched {
                //Send AddedEvent Callback.
                connections[pid] = &host

                fmt.Println("Detected", tunnelCmdline, pid)

                namedPipe := "/tmp/" + strconv.Itoa(int(pid))
                syscall.Mkfifo(namedPipe, 0600)
                stdout, _ := os.OpenFile(namedPipe, os.O_RDONLY, 0600)
                var buff bytes.Buffer
                io.Copy(&buff, stdout)
                stdout.Close()
                fmt.Printf("Payload: %s\n", buff.String())

                if err := json.Unmarshal(buff.Bytes(), &host); err != nil {
                        panic(err)
                }

                fmt.Println("Host: ", host)

                a(pid, host)

            }
        }
    }
    delete(processing, pid)
}

func RemoveConnection(ev * psnotify.ProcEventExit, r ConnRemovedEvent) {
    pid := int32(ev.Pid)
    
    h, ok := connections[pid]
    if ok {
        delete(connections, pid)
        //Send RemoveEvent Callback
        r(pid, *h)
    }
}

func handleEvents(uid int, pid int, a ConnAddedEvent, r ConnRemovedEvent) {

    // New Process Watcher. 
    watcher, err := psnotify.NewWatcher()
    if err != nil {
        panic(err)
    }

    // Process fork, exec, exit & error events
    go func() {
        for {
            select {
            case ev := <-watcher.Exec:
                go AddConnection(ev, uid, a)
            case ev := <-watcher.Exit:
                go RemoveConnection(ev, r)
            }
        }
    }()

    // Process fork, exec, exit & error events
    err = watcher.Watch(pid, psnotify.PROC_EVENT_EXEC | psnotify.PROC_EVENT_EXIT)
    if err != nil {
        fmt.Println(err)
    }
}

func Monitor(uid int, a ConnAddedEvent, r ConnRemovedEvent) {
    
    connections = make(map[int32]*Host)
    processing  = make(map[int32]bool)
    
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
                handleEvents(uid, int(pids[pid]), a, r)
            }            
        }
    }
}