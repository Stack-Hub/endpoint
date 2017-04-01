package server

import (
    "fmt"
    "regexp"
    "encoding/json"
    "os"
    "os/exec"
    "strconv"
    "bytes"
    "io"
    "net"
    "log"
    "bufio"
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

var connections map[int32]*Host
var processing map[int32]bool

func check(e error) {
    if e != nil {
        fmt.Fprintln(os.Stderr, e)
        panic(e)
    }
}


type ConnAddedEvent   func(p int32, h Host)
type ConnRemovedEvent func(p int32, h Host)
 
func RemoveConnection(h *Host, r ConnRemovedEvent) {
    pid := h.Pid
    
    h, ok := connections[pid]
    if ok {
        delete(connections, pid)
        //Send RemoveEvent Callback
        r(pid, *h)
    }
}

func conntrack(h *Host, r ConnRemovedEvent) {
    // Delete user
	cmdName := "conntrack"
	cmdArgs := []string{"-E", "-e", "UPDATE", "-p", "tcp", "--state", "FIN_WAIT", "--orig-port-src", strconv.Itoa(int(h.RemotePort)), "--orig-src", h.RemoteIP}
    
    cmd := exec.Command(cmdName, cmdArgs...)

    stdout, err := cmd.StdoutPipe()
    check(err)

    scanner := bufio.NewScanner(stdout)
	go func() {
        mstr := ` \[UPDATE\] tcp      .* .* FIN_WAIT src=.* dst=.* sport=%d dport=.* src=%s dst=.* sport=.* dport=.*`
        mstr = fmt.Sprintf(mstr, h.RemotePort, h.RemoteIP)
        fmt.Println(mstr)
        
		for scanner.Scan() {
            ev := scanner.Text()
            fmt.Println(ev)
            matched, _ := regexp.MatchString(mstr, ev)
            if matched {    
                RemoveConnection(h, r)
                cmd.Process.Kill()
            }
		}
	}()
    
    
    if err = cmd.Start(); err != nil { //Use start, not run
        check(err)
    }

    _ = cmd.Wait()
}

func waitForClose(pid int) bool {
    // Add user
	cmdName := "flock"
    cmdArgs := []string{"/tmp/" + strconv.Itoa(pid), "-c", "echo done"}
    
    out, err := exec.Command(cmdName, cmdArgs...).Output()
    if err != nil {
        return false
    }
    fmt.Println(out)
    
    return true
}


func handleEvents(c net.Conn, uid int, pid int, a ConnAddedEvent, r ConnRemovedEvent) {
    //Declare host to store connection information
    var host Host
    var buff bytes.Buffer
    io.Copy(&buff, c)
    c.Close()
    fmt.Printf("Payload: %s\n", buff.String())

    if err := json.Unmarshal(buff.Bytes(), &host); err != nil {
            panic(err)
    }

    fmt.Println("Host: ", host)

    //Send AddedEvent Callback.
    connections[host.Pid] = &host
    a(host.Pid, host)
    
    if waitForClose(int(host.Pid)) == true {
        RemoveConnection(&host, r)
    }
}

func Monitor(uid int, a ConnAddedEvent, r ConnRemovedEvent) {
    connections = make(map[int32]*Host)
    processing  = make(map[int32]bool)
    
    // Get the list of all Pids in the system 
    // and search for sshd process.
    p := os.Getpid()
    f := "/tmp/" + strconv.Itoa(p) + ".sock"
    l, _ := net.Listen("unix", f)
    fmt.Printf("Waiting for Connections\n")

    for {
		fd, err := l.Accept()
		if err != nil {
			log.Fatal(err)
			return
		}
		go handleEvents(fd, uid, 1, a, r)
	}    
}