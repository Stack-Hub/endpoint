/**
 * User package to add and remove linux user with ForceCommand
 * 
 * Supports both password based & key based user authenticaion.
 */
package user

import (
	"fmt"
    "regexp"
    "errors"
    "os/exec"
    "os/user"
    "os"
    "io"
    "io/ioutil"
    "strconv"
    "strings"
)

const (
    PASSWD      = 0
    KEY         = 1
    SSHD_CONFIG = "/etc/ssh/sshd_config" 
    MATCHBLK    = `
Match User %s
    AllowTCPForwarding yes
    X11Forwarding no
    AllowAgentForwarding no
    PermitTTY no
    ForceCommand %s -t $SSH_ORIGINAL_COMMAND
`

)

type User struct {
    Name string
    Uid  int
    mode int
}

func check(e error) {
    if e != nil {
        fmt.Fprintln(os.Stderr, e)
        panic(e)
    }
}

func NewUserWithPassword(prefix string, pass string) *User {
    username, err := addUniqueUser(prefix)
    check(err)
    
    uid, err := user.Lookup(username)
    check(err)
    
    u := &User {username, uid, PASSWD}
    
    /**
     * Recover in case of any panic down the stack, 
     * delete user and return nil
     */
    defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovered in f", r)
            u.Delete()
            u = nil
        }
    }()
    
    setUserPasswd(username, pass)
    addUserSSHDConfig(SSHD_CONFIG, username)
    restartSSHServer()
    
    return u
}

func NewUserWithKey(prefix string, keyfile string) *User {
    username, err := addUniqueUser(prefix)
    check(err)

    uid, err := user.Lookup(username)
    check(err)
        
    u := &User {username, uid, KEY}
    
    /**
     * Recover in case of any panic down the stack, 
     * delete user and return nil
     */
    defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovered in f", r)
            u.Delete()
            u = nil
        }
    }()

    allowKeyAccess(username, keyfile)
    
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
    fmt.Println(string(out))
    
    if (u.mode == PASSWD) {
        removeUserSSHDConfig(SSHD_CONFIG, u.Name)
        restartSSHServer()
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
      matchBlkStr := fmt.Sprintf(MATCHBLK, username, os.Args[0])

      f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
      check(err)
      defer f.Close()

      _, err = f.WriteString(matchBlkStr)
      check(err)

      return nil
}

/**
 * Remove Match block from sshd config file
 *
 * path: the path of the config file
 * username: Username for Match block.
 */
func removeUserSSHDConfig(path, username string) error {    
      matchBlkStr := fmt.Sprintf(MATCHBLK, username, os.Args[0])

      input, err := ioutil.ReadFile(path)
      if err != nil {
              check(err)
      }
    
      config := strings.Replace(string(input), matchBlkStr, "", -1)
    
      err = ioutil.WriteFile(path, []byte(config), 0644)
      if err != nil {
              check(err)
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
    fmt.Println(string(out))
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
    fmt.Println(string(out))
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
* Check if User exists
*/
func chkUser(username string) error {
    // Add user
	cmdName := "id"
	cmdArgs := []string{"-u", username}
    
    out, err := exec.Command(cmdName, cmdArgs...).Output()
    fmt.Println("chkuser ", string(out))
    return err
}

/**
* Add User without Password
*/
func addUser(username string) error {
    // Add user
	cmdName := "adduser"
	cmdArgs := []string{"--disabled-password", "--gecos", "\"" + username + "\"", username}
    
    out, err := exec.Command(cmdName, cmdArgs...).Output()
    fmt.Println("adduser ", string(out))
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
    check(err)

    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err = cmd.Start(); err != nil { //Use start, not run
        check(err)
    }

    io.WriteString(stdin, username + ":" + passwd)
    stdin.Close()
    cmd.Wait()
    return nil
}

/**
* Generate Unique Username based on give prefix
* and add user in linux system
*/
func addUniqueUser(prefix string) (string, error) {
    username := ""
    for id := 1; id < 1000; id++ {
        username = prefix + strconv.Itoa(id) 
        
        err := chkUser(username)
        if err == nil {
            fmt.Println(err)
            continue
        }
        
        addUser(username)
        return username, nil
    }
    
    return "", errors.New("Could not add user with prefix " + prefix)
}

/**
* Parse user@host string and return user & host 
*/
func parsePath(path string) (string, string, error) {
    var remotePathRegexp = regexp.MustCompile("^((([^@]+)@)([^:]+))$")
	parts := remotePathRegexp.FindStringSubmatch(path)
	if len(parts) == 0 {
		return "", "", errors.New(fmt.Sprintf("Could not parse remote path: [%s]\n", path))
	}
    return parts[3], parts[4], nil
}

/**
* Add authorized key for the user with force command.
*/
func allowKeyAccess(username string, keyFile string) {
    forceCommand := `command="%s -t $SSH_ORIGINAL_COMMAND",no-X11-forwarding,no-pty,no-agent-forwarding  %s`
    
    // Make home Directory path
    homeDir := "/home/" + username 
        
    key, err := ioutil.ReadFile(keyFile)
    check(err) 
    
    entry := fmt.Sprintf(forceCommand, os.Args[0], key)

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

