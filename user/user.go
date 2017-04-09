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
 *
*
 * User package to add and remove linux user with ForceCommand
 * 
 * Supports both password based & key based user authenticaion.
 */
package user

import (
	"fmt"
    "os/exec"
    "os/user"
    "os"
    "io"
    "io/ioutil"
    "strconv"
    "strings"
    
    "../utils"
    log "github.com/Sirupsen/logrus"
)

const (
    PASSWD      = 0
    KEY         = 1
)

type User struct {
    Name string
    Uid  int
    Mode int
}

/*
 * Map to store created users
 */
var users = make(map[string]*User)

/*
 * Cleanup on termination.
 */
func Cleanup() {
    log.Debug("Deleting Users")
    log.Debug("users = ", users)

    for _, u := range users {
        log.Debug("Deleting ", u)
        u.Delete()
        log.Debug("users = ", users)
    }      
}

func RestoreUser(u *User) {
    users[u.Name] = u
}


func NewUserWithPassword(uname string, pass string) *User {
    log.Debug("uname=", uname, ",pass=", pass)
    err := addUser(uname)
    utils.Check(err)
    
    user, err := user.Lookup(uname)
    utils.Check(err)
    
    uid, err := strconv.Atoi(user.Uid)
    utils.Check(err)
    
    u := &User {uname, uid, PASSWD}

    log.Debug("Adding user ", u)
    // Store user info for cleanup
    users[uname] = u    
    log.Debug("users = ", users)

    
    setUserPasswd(uname, pass)
    addUserSSHDConfig(utils.SSHD_CONFIG, uname)
    restartSSHServer()
    
    return u
}

func NewUserWithKey(uname string, keyfile string) *User {
    err := addUser(uname)
    utils.Check(err)

    user, err := user.Lookup(uname)
    utils.Check(err)
        
    uid, err := strconv.Atoi(user.Uid)
    utils.Check(err)
    
    u := &User {uname, uid, KEY}
    
    // Store user info for cleanup
    users[uname] = u    
    
    allowKeyAccess(uname, keyfile)
    
    return u
}

/**
 * Delete User from system
 */
func (u *User) Delete() error {
    // Delete user
	cmdName := "deluser"
	cmdArgs := []string{"--remove-home", u.Name}
    
    out, err := exec.Command(cmdName, cmdArgs...).Output()
    log.Debug(string(out))
    
    if (u.Mode == PASSWD) {
        removeUserSSHDConfig(utils.SSHD_CONFIG, u.Name)
        restartSSHServer()
    }
    
    // Remove user from map store
    if err == nil {
        delete(users, u.Name)
    }
    
    return err
}

/**
 * Add Match block to the end of sshd config file
 *
 * path: the path of the config file
 * username: Username for Match block.
 */
func addUserSSHDConfig(path, username string) error {    
      matchBlkStr := fmt.Sprintf(utils.MATCHBLK, username)

      f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
      utils.Check(err)
      defer f.Close()

      _, err = f.WriteString(matchBlkStr)
      utils.Check(err)

      return nil
}

/**
 * Remove Match block from sshd config file
 *
 * path: the path of the config file
 * username: Username for Match block.
 */
func removeUserSSHDConfig(path, username string) error {    
      matchBlkStr := fmt.Sprintf(utils.MATCHBLK, username)

      input, err := ioutil.ReadFile(path)
      if err != nil {
          utils.Check(err)
      }
    
      config := strings.Replace(string(input), matchBlkStr, "", -1)
    
      err = ioutil.WriteFile(path, []byte(config), 0644)
      if err != nil {
          utils.Check(err)
      }
      
      return nil
}

/**
 * Restart SSH Server
 */
func restartSSHServer() error {    
      
	cmdName := "service"
    cmdArgs := []string{"ssh", "restart"}
    
    out, err := exec.Command(cmdName, cmdArgs...).Output()
    log.Debug(string(out))
    return err
}

/**
 * Chown based on username
 */
func chown(username string, file string) error {

    // Add user
	cmdName := "chown"
    cmdArgs := []string{username + ":" + username, file}
    
    out, err := exec.Command(cmdName, cmdArgs...).Output()
    log.Debug(string(out))
    return err
}

/**
* Check whether the given file or directory exists or not
*/
func exists(path string) (bool, error) {
    _, err := os.Stat(path)
    if err == nil { return true, nil }
    if os.IsNotExist(err) { return false, nil }
    return true, err
}

/**
* Add User without Password
*/
func addUser(username string) error {
    // Add user
	cmd  := "useradd"
	args := []string{username}
    log.Debug("cmd=", cmd, ",args=", args)
    
    c := exec.Command(cmd, args...)
    c.Stderr = os.Stderr
    err := c.Run()
    
    return err
}

/**
* Set Password for User
*/
func setUserPasswd(username string, passwd string) error {
    cmdName := "chpasswd"
    cmdArgs := []string{}

    cmd := exec.Command(cmdName, cmdArgs...)

    stdin, err := cmd.StdinPipe()
    utils.Check(err)

    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err = cmd.Start(); err != nil { //Use start, not run
        utils.Check(err)
    }

    io.WriteString(stdin, username + ":" + passwd)
    stdin.Close()
    cmd.Wait()
    return nil
}

/**
* Add authorized key for the user with force command.
*/
func allowKeyAccess(username string, keyFile string) {
    forceCommand := `command="flock /tmp/$$ -c \"/usr/sbin/trafficrouter -t $SSH_ORIGINAL_COMMAND\",no-X11-forwarding,no-pty,no-agent-forwarding  %s`
    
    // Make home Directory path
    homeDir := "/home/" + username 
        
    key, err := ioutil.ReadFile(keyFile)
    utils.Check(err) 
    
    entry := fmt.Sprintf(forceCommand, key)

    if res, _ := exists(homeDir + "/.ssh"); res != true {
        err = os.Mkdir(homeDir + "/.ssh", 0700)
        utils.Check(err)
        chown(username, homeDir + "/.ssh")
    }

    data := []byte(entry)
    // Path for force command script
    authFile := homeDir + "/.ssh/authorized_keys" 
    err = ioutil.WriteFile(authFile, data, 0600)
    utils.Check(err)

    err = chown(username, authFile)
    utils.Check(err)
}

