package reverse_tunnel

import (
	"bufio"
	"fmt"
	"os"
    "regexp"
	"os/exec"
)


func Start(privKeyFile string, username string, hostname string, port string) (string, *exec.Cmd, error)  {
	// ssh open reverse tunnel
	cmdName := "ssh"
	cmdArgs := []string{"-q", "-T", "-i", privKeyFile, "-o", "StrictHostkeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-R", "0:localhost:" + port, username + "@" + hostname, "detach"}
    
	cmd := exec.Command(cmdName, cmdArgs...)
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating StdoutPipe for ssh (revers tunnel)", err)
		os.Exit(1)
	}
        
    var URLRegexp = regexp.MustCompile("^(.*)?(http(s)?://.*:[0-9]+)$")
    channel := make(chan string)

    err = cmd.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error starting Cmd", err)
		os.Exit(1)
	}
    
	// Extract URL from ssh output
	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
            // Scan for URL in the output
            parts := URLRegexp.FindStringSubmatch(scanner.Text())
            if len(parts) != 0 {
                channel <- parts[2]
            }
		}
	}()
    
    // Wait for go routine to send URL
    url := <-channel
    return url, cmd, err
}

func Stop(cmd *exec.Cmd) (error) {
 
    err := cmd.Process.Kill()
	if err != nil {
        fmt.Fprintln(os.Stderr, "Error killing ssh (revers tunnel) process", err)
        return err
	}
    return nil
}
