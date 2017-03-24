package user

import (
	"fmt"
    "regexp"
    "errors"
    "os/exec"
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
    MATCHBLK    = `Match User %s
      AllowTCPForwarding yes
      X11Forwarding no
      AllowAgentForwarding no
      PermitTTY yes
      ForceCommand sleep infinity`

)

type User struct {
    Name string
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
    
    user := User {username, PASSWD}
    setUserPasswd(username, pass)
    addUserSSHDConfig(SSHD_CONFIG, username)
    
    return &user
}

func NewUserWithKey(prefix string, keyfile string) *User {
    username, err := addUniqueUser(prefix)
    check(err)

    user := User {username, KEY}
    allowKeyAccess(username, keyfile)
    
    return &user
}

func (u *User) Delete() error {
    // Delete user
	cmdName := "deluser"
	cmdArgs := []string{"--remove-home", u.Name}
    
    out, err := exec.Command(cmdName, cmdArgs...).Output()
    fmt.Println(string(out))
    
    if (u.mode == PASSWD) {
        removeUserSSHDConfig(SSHD_CONFIG, u.Name)
    }
    return err
}

/**
 * Add Match block to the end of sshd config file
 *
 * path: the path of the file
 * username: Username for Match block.
 */
func addUserSSHDConfig(path, username string) error {    
      matchBlkStr := fmt.Sprintf(MATCHBLK, username)

      f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
      if err != nil {
              return err
      }
      defer f.Close()

      _, err = f.WriteString(matchBlkStr)
      if err != nil {
              return err
      }
      return nil
}

/**
 * Add Match block to the end of sshd config file
 *
 * path: the path of the file
 * username: Username for Match block.
 */
func removeUserSSHDConfig(path, username string) error {    
      matchBlkStr := fmt.Sprintf(MATCHBLK, username)

      input, err := ioutil.ReadFile("myfile")
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

func chown(username string, file string) error {

    // Add user
	cmdName := "chown"
    cmdArgs := []string{username + ":" + username, file}
    
    out, err := exec.Command(cmdName, cmdArgs...).Output()
    fmt.Println(string(out))
    return err
}

// exists returns whether the given file or directory exists or not
func exists(path string) (bool, error) {
    _, err := os.Stat(path)
    if err == nil { return true, nil }
    if os.IsNotExist(err) { return false, nil }
    return true, err
}

func chkUser(username string) error {
    // Add user
	cmdName := "id"
	cmdArgs := []string{"-u", username}
    
    out, err := exec.Command(cmdName, cmdArgs...).Output()
    fmt.Println(string(out))
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

func setUserPasswd(username string, passwd string) error {
    cmdName := "chpasswd"
    cmdArgs := []string{}

    cmd := exec.Command(cmdName, cmdArgs...)

    stdin, err := cmd.StdinPipe()
    if err != nil {
        fmt.Println(err)
    }

    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err = cmd.Start(); err != nil { //Use start, not run
        fmt.Println("An error occured: ", err) //replace with logger, or anything you want
        return err
    }

    io.WriteString(stdin, username + ":" + passwd)
    stdin.Close()
    cmd.Wait()
    return nil
}


func addUniqueUser(prefix string) (string, error) {
    username := ""
    for id := 1; id < 1000; id++ {
        username = prefix + strconv.Itoa(id) 
        
        err := chkUser(username)
        if err == nil {
            continue
        } else {
            return "", err
        }
        
        addUser(username)
        break
    }
    
    return username, nil
}


func parsePath(path string) (string, string, error) {
    var remotePathRegexp = regexp.MustCompile("^((([^@]+)@)([^:]+))$")
	parts := remotePathRegexp.FindStringSubmatch(path)
	if len(parts) == 0 {
		return "", "", errors.New(fmt.Sprintf("Could not parse remote path: [%s]\n", path))
	}
    return parts[3], parts[4], nil
}

func allowKeyAccess(username string, keyFile string) {
    forceCommand := `command="sleep infinity $SSH_ORIGINAL_COMMAND",no-X11-forwarding,pty,no-agent-forwarding  %s`
    
    // Make home Directory path
    homeDir := "/home/" + username 
        
    key, err := ioutil.ReadFile(keyFile)
    check(err) 
    
    entry := fmt.Sprintf(forceCommand, key)

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

