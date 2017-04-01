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
    "syscall"
    
    "./omap"
    "./client"
    "./forcecmd"
    "./user"
    "./server"
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
        port := strconv.Itoa(int(h.Value.(server.Host).ListenPort))
        out, _ := net.Dial("tcp", "127.0.0.1:" + port)
        go io.Copy(out, in)
        io.Copy(in, out)
        defer out.Close()    
    }
}

func ConnAddEv(p int32, h server.Host) {
    m.Add(p, h)
    fmt.Printf("Connected %s:%d on Port %d\n", 
               h.RemoteIP, 
               h.RemotePort, 
               h.ListenPort)
    
}

func ConnRemoveEv(p int32, h server.Host) {
    m.Remove(p)
    fmt.Printf("Removed %s:%d from Port %d\n", 
               h.RemoteIP, 
               h.RemotePort, 
               h.ListenPort)
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
    clientArg := flag.String("c", "", "Server Address in user@host Format")

    // Server mode options
//    public := flag.String("pub", "", "Public Key File valid only in Server mode")
    serverArg := flag.Bool("s", false, "Run as Server")
    username := flag.String("u", "", "Username for Server")
    uid := flag.Int("uid", -1, "User ID")
    mode := flag.Int("mode", -2, "User Create mode")

    tnl := flag.Bool("t", false, "Tunnel command used as SSH Force Command")

    //Parse Command lines
    flag.Parse()
    tail := flag.Args()  
    
    if (*clientArg != "") {
        port := tail[0]
        user, hostname, err := parsePath(*clientArg)
        check(err)
        
        cmd, err := client.Start(*private, user, hostname, port)
        defer client.Stop(cmd)
        fmt.Println(cmd)
        blockForever()
        
    } else if (*serverArg == true) {
        fmt.Println(os.Args)
        if (*username == "") {
            u := user.NewUserWithPassword("tr", "1234567890")
            
            args := make([]string, len(os.Args) + 6)
            copy(args, os.Args)
            args[len(os.Args)]     = "-u"
            args[len(os.Args) + 1] = u.Name
            args[len(os.Args) + 2] = "-uid"
            args[len(os.Args) + 3] = strconv.Itoa(u.Uid)
            args[len(os.Args) + 4] = "-mode"
            args[len(os.Args) + 5] = strconv.Itoa(u.Mode)
            
            fmt.Println("Swamping new exec")
            err := syscall.Exec("/usr/sbin/" + os.Args[0], args, os.Environ())
            fmt.Println("Swamped", err)
        }
        
        u := &user.User{Name: *username, Uid: *uid, Mode: *mode}
        
        m = omap.New()
        server.Monitor(u.Uid, ConnAddEv, ConnRemoveEv)

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
        forcecmd.SendConfig()
    }

}