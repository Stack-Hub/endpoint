package main

import (
    "fmt"
    "net"
    "os"
    "io"
    "strconv"
    "flag"
    "regexp"
    "errors"
    "runtime"
    
    ssh "./connmonitor" 
    "./omap"
    "./reverse_tunnel"
    "./tunnel"
    "./user"
)

const (
    SERVER_HOST = "0.0.0.0"
    SERVER_PORT = "80"
    SERVER_TYPE = "tcp"
)

var m * omap.OMap

func check(e error) {
    if e != nil {
        fmt.Fprintln(os.Stderr, e)
        panic(e)
    }
}

func blockForever() {
    for {
        runtime.Gosched()
    }
}

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
    fmt.Printf("Connected %s:%d on Port %d\n", 
               h.RemoteIP, 
               h.RemotePort, 
               h.LocalPort)
    
}

func ConnRemoveEv(p int32, h ssh.Host) {
    m.Remove(p)
    fmt.Printf("Removed %s:%d from Port %d\n", 
               h.RemoteIP, 
               h.RemotePort, 
               h.LocalPort)
}


func parsePath(path string) (string, string, error) {
    var remotePathRegexp = regexp.MustCompile("^((([^@]+)@)([^:]+))$")
	parts := remotePathRegexp.FindStringSubmatch(path)
	if len(parts) == 0 {
		return "", "", errors.New(fmt.Sprintf("Could not parse remote path: [%s]\n", path))
	}
    return parts[3], parts[4], nil
}


func main() {

    // Command line options
    // Client mode options
    private := flag.String("priv", "", "Private Key File valid only in Client mode")
    client := flag.String("c", "", "Server Address in user@host Format")

    // Server mode options
//    public := flag.String("pub", "", "Public Key File valid only in Server mode")
    server := flag.Bool("s", false, "Run as Server")
//    username := flag.String("u", "", "Username for Server")

    tnl := flag.Bool("t", false, "Tunnel command used as SSH Force Command")

    //Parse Command lines
    flag.Parse()
    tail := flag.Args()  
    
    if (*client != "") {
        port := tail[0]
        user, hostname, err := parsePath(*client)
        check(err)
        
        cmd, err := reverse_tunnel.Start(*private, user, hostname, port)
        defer reverse_tunnel.Stop(cmd)
        fmt.Println(cmd)
        blockForever()
        
    } else if (*server == true) {
        m = omap.New()
        u := user.NewUserWithPassword("tr", "1234567890")
        
        ssh.Monitor(u.Uid, ConnAddEv, ConnRemoveEv)

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
    } else if (*tnl == true) {
        tunnel.SendConfig()
    }

}