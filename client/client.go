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
    "os"
	"os/exec"
    "log"
    
    "../utils"
)


func StartWithKey(key string, u string, h string, p string) *exec.Cmd  {
	// ssh open reverse tunnel
	cmd := "ssh"
	args := []string{"-q", 
                     "-t", 
                     "-i", key, 
                     "-o", "StrictHostkeyChecking=no", 
                     "-o", "UserKnownHostsFile=/dev/null", 
                     "-R", "0:localhost:" + p, u + "@" + h, 
                     "{\"port\":" + p + "}"}
    
    c := exec.Command(cmd, args...)
    c.Stdout = os.Stdout
    c.Stderr = os.Stderr
    err := c.Start()
    utils.Check(err)
        
    return c
}

func StartWithPasswd(u string, pass string, h string, p string) *exec.Cmd {
	// ssh open reverse tunnel
	cmd := "sshpass"
	args := []string{"-p", pass,
                     "ssh",
                     "-q", 
                     "-t", 
                     "-o", "StrictHostkeyChecking=no", 
                     "-o", "UserKnownHostsFile=/dev/null", 
                     "-R", "0:localhost:" + p, u + "@" + h, 
                     "{\"port\":" + p + "}"}

    log.Println(cmd, args)
    
	c := exec.Command(cmd, args...)
    c.Stdout = os.Stdout
    c.Stderr = os.Stderr
    err := c.Start()
    utils.Check(err)
    
    return c
}


func Stop(cmd *exec.Cmd) {
 
    err := cmd.Process.Kill()
    utils.Check(err)
}
