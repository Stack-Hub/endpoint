package main

import (
	"fmt"
    "flag"
    "regexp"
    "errors"
    "os/exec"
    "os"
    "io/ioutil"

    "./reverse_tunnel"
)

func check(e error) {
    if e != nil {
        fmt.Fprintln(os.Stderr, e)
        panic(e)
    }
}

const executable = "reverse_tunnel"

func chown(username string, file string) error {

    // Add user
	cmdName := "chown"
    cmdArgs := []string{username + ":" + username, file}
    
    out, err := exec.Command(cmdName, cmdArgs...).Output()
    fmt.Println(string(out))
    return err
}

func allowAccess(username string, keyFile string) {
    forceCommand := `command="%s",no-X11-forwarding,no-pty,no-agent-forwarding  %s`
    
    // Make home Directory path
    homeDir := "/home/" + username 
    
    // Path for force command script
    path := homeDir+ "/" + executable
    
    key, err := ioutil.ReadFile(keyFile)
    check(err) 
    
    entry := fmt.Sprintf(forceCommand, path, key)

    if res, _ := exists(homeDir + "/.ssh"); res != true {
        err = os.Mkdir(homeDir + "/.ssh", 0700)
        check(err)
        chown(username, homeDir + "/.ssh")
    }

    data := []byte(entry)
    // Path for force command script
    authFile := homeDir + "/.ssh/authorized_keys" 
    err = ioutil.WriteFile(authFile, data, 0600)
    check(err)

    err = chown(username, authFile)
    check(err)
}

// exists returns whether the given file or directory exists or not
func exists(path string) (bool, error) {
    _, err := os.Stat(path)
    if err == nil { return true, nil }
    if os.IsNotExist(err) { return false, nil }
    return true, err
}

func writeScript(username string) error {
    // Shell Script that should be executed when connection is opened
    script := `#!/bin/bash

    # Get SERVER IP
    SERVERIP=$(ip route get 8.8.8.8 | awk 'NR==1 {print $NF}')
    DOMAIN="dupper.co"

    # Get opened port
    PORT=$(sudo lsof -i -P -n  -p $PPID | grep $PPID | egrep .*IPv4.*LISTEN | sed -E 's/.*IPv4.*:([0-9]+) \(LISTEN\)/\1/g')

    # Inform user
    echo "Your port is shared at http://$(curl -s http://169.254.169.254/latest/meta-data/public-hostname):${PORT}"
    echo "Press Ctrl + D to stop sharing."

    if [[ ${SSH_ORIGINAL_COMMAND} == "detach" ]]; then
        tail -F /tmp/nonexistent 2>/dev/null >/dev/null
    else
        cat
    fi`
    
    data := []byte(script)
    // Path for force command script
    path := "/home/" + username + "/" + executable
    err := ioutil.WriteFile(path, data, 0777)
    return err
    
}

func addUser(username string) error {
    // Add user
	cmdName := "adduser"
	cmdArgs := []string{"--disabled-password", "--gecos", "\"" + username + "\"", username}
    
    out, err := exec.Command(cmdName, cmdArgs...).Output()
    fmt.Println(string(out))
    return err
}

func delUser(username string) error {
    // Delete user
	cmdName := "deluser"
	cmdArgs := []string{"--remove-home", username}
    
    out, err := exec.Command(cmdName, cmdArgs...).Output()
    fmt.Println(string(out))
    return err
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
    public := flag.String("pub", "", "Public Key File valid only in Server mode")
    server := flag.Bool("s", false, "Run as Server")
    username := flag.String("u", "", "Username for Server")

    //Parse Command lines
    flag.Parse()
    tail := flag.Args()  
    
    if (*client != "") {
        port := tail[0]
        user, hostname, err := parsePath(*client)
        check(err)
        
        url, cmd, err := reverse_tunnel.Start(*private, user, hostname, port)
        defer reverse_tunnel.Stop(cmd)
        fmt.Println(url, cmd)
    } else if (*server == true) {
        err := addUser(*username)
        check(err)
        defer delUser(*username)
        
        // Write script at Path
        err = writeScript(*username)
        check(err)
        
        allowAccess(*username, *public)
        
        var wait int
        fmt.Scanf("%d", wait)
    }
}
