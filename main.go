package main

import (
    "fmt"
    "net"
    "os"
    "io"
    "strconv"
    
    ssh "./connmonitor" 
    "./omap"
)

const (
    SERVER_HOST = "0.0.0.0"
    SERVER_PORT = "80"
    SERVER_TYPE = "tcp"
)

var m * omap.OMap

// Handles incoming requests.
func handleRequest(in net.Conn) {
    defer in.Close()

    h := m.Next()
    if h == nil {
        // Send a response back to person contacting us.
        in.Write([]byte("No Routes available."))   
    } else {
        port := strconv.Itoa(int(h.Value.(ssh.Host).LocalPort))
        out, _ := net.Dial("tcp", "127.0.0.1:" + port)
        go io.Copy(out, in)
        io.Copy(in, out)
        defer out.Close()    
    }
}

func ConnAddEv(p int32, h ssh.Host) {
    m.Add(p, h)
}

func ConnRemoveEv(p int32) {
    m.Remove(p)
}


func main() {
    
    m = omap.New()
    
    ssh.Monitor(ConnAddEv, ConnRemoveEv)
    
    l, err := net.Listen(SERVER_TYPE, SERVER_HOST+":"+SERVER_PORT)
    if err != nil {
        fmt.Println("Error listening:", err.Error())
        os.Exit(1)
    }
    // Close the listener when the application closes.
    defer l.Close()
    
    fmt.Println("Listening on " + SERVER_HOST + ":" + SERVER_PORT)
    for {
        // Listen for an incoming connection.
        conn, err := l.Accept()
        if err != nil {
            fmt.Println("Error accepting: ", err.Error())
            os.Exit(1)
        }
        // Handle connections in a new goroutine.
        go handleRequest(conn)
    }
}