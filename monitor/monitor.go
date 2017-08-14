/* Copyright (C) Ashish Thakwani - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Ashish Thakwani <athakwani@gmail.com>, August 2017
 */
package server

import (
    "encoding/json"
    "os"
    "strconv"
    "bytes"
    "io"
    "net"
    "fmt"
    "syscall"
    
    "github.com/duppercloud/trafficrouter/utils"
    "github.com/duppercloud/trafficrouter/omap"
    log "github.com/Sirupsen/logrus"
)

/*
 *  Map for holding active connections and socket information.
 */
var conns map[*omap.OMap]map[int]*utils.Host = make(map[*omap.OMap]map[int]*utils.Host, 1)
var sockFiles map[*omap.OMap]string = make(map[*omap.OMap]string, 1)

/*
 * Cleanup on termination.
 */
func Cleanup() {
    for _, m := range conns {
        for _, h := range m {
            log.Debug("Server killing ", h.Pid)
            syscall.Kill(h.Pid, syscall.SIGINT)
            os.Remove(utils.RUNPATH + strconv.Itoa(int(h.Pid)))
        }
    }
    
    for _, f := range sockFiles {
        log.Debug("Removing ", f)
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
    // Blocking flock on pid file.
    mode := utils.LOCK_EX
    filename := utils.RUNPATH + strconv.Itoa(p)
    
    _, err := utils.LockFile(filename, true, mode)
    if err == nil {
        return true
    }
    
    return false
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
    
    log.Debug("Payload: %s", b.String())

    err := json.Unmarshal(b.Bytes(), &h) 
    utils.Check(err)
    
    go addConnection(m, uname, &h, a)
    
    if waitForClose(int(h.Pid)) == true {
        go removeConnection(m, uname, &h, r)
    }
}

/*
 *  Monitor incoming connections and invoke callback
 *  when client is added or removed.
 */
func Monitor(m *omap.OMap, uname string, a ConnAddedEvent, r ConnRemovedEvent) {    
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