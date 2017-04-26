/* Copyright 2017, Ashish Thakwani. 
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.LICENSE file.
 */
package server

import (
    "encoding/json"
    "os"
    "os/exec"
    "strconv"
    "bytes"
    "io"
    "net"
    "fmt"
    "syscall"
    
    "../utils"
    "../omap"
    log "github.com/Sirupsen/logrus"
)

/*
 *  Map for holding active connections and socket information.
 */
var conns map[*omap.OMap]map[int]*utils.Host
var sockFiles map[*omap.OMap]string

/*
 * Cleanup on termination.
 */
func Cleanup() {
    for _, m := range conns {
        for _, h := range m {
            log.Debug("Server killing ", h.Pid)
            syscall.Kill(h.Pid, syscall.SIGINT)
        }
    }
    
    for _, f := range sockFiles {
        os.Remove(f)
    }
}


/*
 *  Event callback for connection add & remove.
 */
type ConnAddedEvent   func(m *omap.OMap, uname string, p int, h *utils.Host)
type ConnRemovedEvent func(m *omap.OMap, uname string, p int, h *utils.Host)

/*
 *  Add connection to map and invoke callback.
 */
func addConnection(m *omap.OMap, uname string, h *utils.Host, a ConnAddedEvent) {
    p := h.Pid
    
    if conns[m] == nil {
        conns[m] = make(map[int]*utils.Host, 1)
    }
    conns[m][p] = h

    //Send AddedEvent Callback.
    a(m, uname, p, h)
}

/*
 *  Remove connection from map and invoke callback.
 */
func removeConnection(m *omap.OMap, uname string, h *utils.Host, r ConnRemovedEvent) {
    p := h.Pid
    
    h, ok := conns[m][p]
    if ok {
        delete(conns[m], p)
        
        //Send RemoveEvent Callback
        r(m, uname, p, h)
    }
}


/*
 *  Wait for lock file to be released.
 */
func waitForClose(p int) bool {
    // Add user
	cmd := "flock"
    args := []string{utils.RUNPATH + strconv.Itoa(p), "-c", "echo done"}
    
    _, err := exec.Command(cmd, args...).Output()
    if err != nil {
        return false
    }
    
    return true
}

/*
 *  Handle Socket connection
 */
func handleClient(c net.Conn, m *omap.OMap, uname string, a ConnAddedEvent, r ConnRemovedEvent) {
    var h utils.Host
    var b bytes.Buffer
    
    // Copy socket data to buffer
    io.Copy(&b, c)
    c.Close()
    
    log.Printf("Payload: %s\n", b.String())

    err := json.Unmarshal(b.Bytes(), &h) 
    utils.Check(err)
    
    addConnection(m, uname, &h, a)
    
    if waitForClose(int(h.Pid)) == true {
        removeConnection(m, uname, &h, r)
    }
}

/*
 *  Monitor incoming connections and invoke callback
 *  when client is added or removed.
 */
func Monitor(m *omap.OMap, uname string, a ConnAddedEvent, r ConnRemovedEvent) {
    // Initialize connections map to store active connections.
    conns = make(map[*omap.OMap]map[int]*utils.Host, 1)
    sockFiles = make(map[*omap.OMap]string, 1)
    
    // Get process pid and open unix socket.
    f := utils.RUNPATH + uname + ".sock"
    sockFiles[m] = f
    
    l, err := net.Listen("unix", f)
    utils.Check(err)
    fmt.Println("Waiting for Connections on ", f)

    for {
        // Handle incoming conection.
        fd, err := l.Accept()
        utils.Check(err)

        go handleClient(fd, m, uname, a, r)
    }
    

}