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
package client

import (
	"fmt"
	"os"
	"os/exec"
)


func Start(privKeyFile string, username string, hostname string, port string) (*exec.Cmd, error)  {
	// ssh open reverse tunnel
	cmdName := "ssh"
	cmdArgs := []string{"-q", "-t", "-i", privKeyFile, "-o", "StrictHostkeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-R", "0:localhost:" + port, username + "@" + hostname, "{\"port\":" + port + "}"}
    
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
