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
    "syscall"
    "math/rand"
    "time"
    "log"
    
    "./omap"
    "./client"
    "./forcecmd"
    "./user"
    "./server"
    "./utils"
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
        port := strconv.Itoa(int(h.Value.(utils.Host).ListenPort))
        out, _ := net.Dial("tcp", "127.0.0.1:" + port)
        go io.Copy(out, in)
        io.Copy(in, out)
        defer out.Close()    
    }
}

func ConnAddEv(p int, h *utils.Host) {
    m.Add(p, h)
    fmt.Printf("Connected %s:%d on Port %d\n", 
               h.RemoteIP, 
               h.RemotePort, 
               h.ListenPort)
    
}

func ConnRemoveEv(p int, h *utils.Host) {
    m.Remove(p)
    fmt.Printf("Removed %s:%d from Port %d\n", 
               h.RemoteIP, 
               h.RemotePort, 
               h.ListenPort)
}


func parseToken(str string) (string, string, string, string, error) {
    var expr = regexp.MustCompile(`^(((.*)/([^@]+)@)([^:]+):([0-9]+))$`)
	parts := expr.FindStringSubmatch(str)
	if len(str) == 0 {
        utils.Check(errors.New(fmt.Sprintf("Token parse error: [%s]. Format user/passwd@host:port\n", str)))
	}
    return parts[3], parts[4], parts[5], parts[6], nil
}


func init() {
    rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func RandStringRunes(n int) string {
    b := make([]rune, n)
    for i := range b {
        b[i] = letterRunes[rand.Intn(len(letterRunes))]
    }
    return string(b)
}

func externalIP() string {
	ifaces, err := net.Interfaces()
    utils.Check(err)

    for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
        
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
        
		addrs, err := iface.Addrs()
        utils.Check(err)

        for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String()
		}
	}
	return ""
}


func main() {

    /* Command line options
     * Client mode options
     */
    priv  := flag.String("priv", "", "Private key file (client mode)")
    clnt  := flag.String("c",    "", "Server token (ex: user/passwd@host:port) (client mode)")
    cmd   := flag.String("cmd",  "", "Run a single command string through the shell (client mode)")

    /* Server mode options */
    pub  := flag.String("pub",  "",    "Public key file (server mode)")
    srv  := flag.Bool(  "s",    false, "Server mode")
    port := flag.Int(   "p",    80,    "Listening Port (server mode)")
    usr  := flag.String("u",    "",    "Username (server mode)")
    uid  := flag.Int(   "uid",  -1,    "User ID (server mode)")
    mode := flag.Int(   "mode", -2,    "User Create mode (server mode)")

    /* Force Command option */
    frc := flag.Bool("f", false, "Force Command for SSH (tunnel mode)")

    /* Parse Command line */
    flag.Parse()

    if (*clnt != "" || *srv == true || *frc == true) {
        log.Println("priv = ", *priv)
        log.Println("clnt = ", *clnt)
        log.Println("cmd  = ", *cmd)
        log.Println("pub  = ", *pub)
        log.Println("srv  = ", *srv)
        log.Println("port = ", *port)
        log.Println("usr  = ", *usr)
        log.Println("uid  = ", *uid)
        log.Println("mode = ", *mode)
        log.Println("frc  = ", *frc)        
    }
    
    /* Client mode */
    if (*clnt != "") {
        uname, passwd, host, port, err := parseToken(*clnt)
        utils.Check(err)

        log.Println(uname, passwd, host, port)
        
        cmd := client.StartWithPasswd(uname, passwd, host, port)
        defer client.Stop(cmd)
        
        fmt.Println(cmd)
        utils.BlockForever()
        
    } else if (*srv == true) {
        if (*usr == "") {
            passwd := RandStringRunes(10)
            u := user.NewUserWithPassword(utils.DEFAULTUNAME, passwd)

            // Generate token for client.
            fmt.Printf("Sever token - %s/%s@%s:%d\n", u.Name, passwd, externalIP(), *port)

            args := make([]string, len(os.Args) + 6)
            copy(args, os.Args)
            args[len(os.Args)]     = "-u"
            args[len(os.Args) + 1] = u.Name
            args[len(os.Args) + 2] = "-uid"
            args[len(os.Args) + 3] = strconv.Itoa(u.Uid)
            args[len(os.Args) + 4] = "-mode"
            args[len(os.Args) + 5] = strconv.Itoa(u.Mode)
            
            log.Println("Swamping new exec")
            err := syscall.Exec("/usr/sbin/" + os.Args[0], args, os.Environ())
            utils.Check(err)
        }
        
        u := &user.User{Name: *usr, Uid: *uid, Mode: *mode}
        defer u.Delete()
        
        m = omap.New()
        server.Monitor(ConnAddEv, ConnRemoveEv)
        
        addr := fmt.Sprintf("%s:%d", utils.SERVER_HOST, *port)
        
        l, err := net.Listen(utils.SERVER_TYPE, addr)
        utils.Check(err)
        
        // Close the listener when the application closes.
        defer l.Close()

        fmt.Printf("Listening on %s:%d", utils.SERVER_HOST, *port)
        for {
            // Listen for an incoming connection.
            conn, err := l.Accept()
            utils.Check(err)
            
            // Handle connections in a new goroutine.
            go handleRequest(conn)
        }    
    } else if (*frc == true) {
        forcecmd.SendConfig()
    } else {
        flag.PrintDefaults()
    }

}