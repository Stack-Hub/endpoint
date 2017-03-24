package reverse_tunnel

import (
	"fmt"
	"os"
	"os/exec"
)


func Start(privKeyFile string, username string, hostname string, port string) (*exec.Cmd, error)  {
	// ssh open reverse tunnel
	cmdName := "ssh"
	cmdArgs := []string{"-q", "-T", "-i", privKeyFile, "-o", "StrictHostkeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-R", "0:localhost:" + port, username + "@" + hostname, "detach", "{\"port\":" + port + "}"}
    
	cmd := exec.Command(cmdName, cmdArgs...)
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening ssh (revers tunnel)", err)
		os.Exit(1)
	}
        
    fmt.Println(string(out))
    return cmd, err
}

func Stop(cmd *exec.Cmd) (error) {
 
    err := cmd.Process.Kill()
	if err != nil {
        fmt.Fprintln(os.Stderr, "Error killing ssh (revers tunnel) process", err)
        return err
	}
    return nil
}
