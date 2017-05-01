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
    "testing"
    "strconv"
    "fmt"
    "os"
    "os/exec"
    "io/ioutil"
    "runtime"
    "time"
    
    "github.com/duppercloud/trafficrouter/user"
)

const (
    PUBKEY = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDk7Ti7Fb1HmzvtWlDtKclvF9vChVdjp/fkdWdZR26HW54aNIh7YLwC1W8aNd6SUd2PEbUAjH6KujHVxA/dxsuYjQCaNouE+W3D+98UgJrfvG6O444BzUOplcHIUppp06f+utveH1gd3w8eyOQzSmLPTMkhKXvJTRuFgdytnmOh2A2qzE81v7I/ExPiIgdS6uBttFVFUvxBfjUpR6k8KnrmCYscJt4wzBQDPkKeI18K2ZNk8ig5389qlfGW/qRT+bxx0GE2UIaFfDIUL8zKp+KugZs0k1g3vCK/F6OKCWczigjnIoWCEK5txahfyMVv7rBTeK3vIq8X5gL2Lt0PWm1V root@digitalocean`
    PRIVKEY = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA5O04uxW9R5s77VpQ7SnJbxfbwoVXY6f35HVnWUduh1ueGjSI
e2C8AtVvGjXeklHdjxG1AIx+irox1cQP3cbLmI0AmjaLhPltw/vfFICa37xujuOO
Ac1DqZXByFKaadOn/rrb3h9YHd8PHsjkM0piz0zJISl7yU0bhYHcrZ5jodgNqsxP
Nb+yPxMT4iIHUurgbbRVRVL8QX41KUepPCp65gmLHCbeMMwUAz5CniNfCtmTZPIo
Od/PapXxlv6kU/m8cdBhNlCGhXwyFC/MyqfiroGbNJNYN7wivxejiglnM4oI5yKF
ghCubcWoX8jFb+6wU3it7yKvF+YC9i7dD1ptVQIDAQABAoIBAAHEex2mq1F0N2A7
xEgwfGepLX8w/F2+nEFdTrD3xjixUmUtZqhdNNZow3TWWaOTkjxajKU2cHutuFjI
LL8vm77Px+No7GbYbiqHNU+5NnjnwYrE4wHMjesvRtG/IYYTpkZnNu9eGpYQdNNu
BaUHu/+RvjPNWDFTsRS0zflhMa+8MZBl6Juubkim99NIBBx+dZrz9inTIYePlUMF
oiw8JJCTvlju9SW8oLkkOuRs9j89BP5VklUZCihK+rKDVedKD0sV76nTRYfORZae
MH4hmwwzM96XfYCETZc8cdnpEUTto62yDKAcWmGsp3zDi0UlEw7jAVmuwRQxDRbY
Lr/GHXkCgYEA/671W/r4q5opZVIl2Hy06BBlnhYtgympG2v5zQAB6HU4WRY68Em8
QS8/zD3ptUKdv/dpZ/CnocCI53v9mFMbUv2KNo6vgJnLHQYLpOHfItdIPXLVF6Sh
64HkB+nzaxW/IsPhATU5DgVlCurBBdJlUA8mAfB2kwhR/8F14QzMxG8CgYEA5TXI
RmjMGlVaxLJ379fyIB/NpRQu6Awjm/aG7oPXOAVnfpTx9aVnn1cNn4fC73/AQw8l
QHQI/3Tb+CYeLhcl9YXdkPkdzrw8A2dMEXx3NE18HRoZRaFd9LCJENI2x4B+s+Gf
dQqTlUNtLnJ8GE9zAzIT0RXkY2yacpAofo/GtHsCgYA3O0cbSHqhLxsUHQu52S6H
Fsusu6O3Oq+iEdATXZYL7g5vCCNRNsxo1FkWuKUcl7hV+I8Xed/sTBgG0Tz1w7Ya
VlSd9nKo+A/tRBoN0xENiK29QGoRwmmL4zIsF3iSwE7apq+bQDED+1xZYF6z8EAc
bDlMn/ItTtXPxq29ILO3FwKBgQCE3Bvu1Cgay4cFpP1ohR/QBx9IpN5bm024xbmI
39sMmfVXpjZqUSozbl5zLlqMQNzNAiZxqdDdYntu54lu5fQW0TWRJxVkFDAlOOca
666dHpzmsY4ckmDHyNxqZ69hDNZkpk+rpCnPx3muBqZv4P2lyI08ERiFmRoddfpD
AkwHqQKBgB+lglg9YVTavNepvRh5knKTEneVGappbybcN6z1AWQq6Pg53Ouq0WnN
dmwbNzHAFhZtstsdCqWfgu3TRzbhwIUSTtpa/jEgk4ZPO+gRv9nk00nRPvTR17pM
SEU+nRcQdT52CZC1kp3lSvhCKDmXo6+UWWmWy67jebN3QnfVZDn6
-----END RSA PRIVATE KEY-----`
    PASSWD  = 1
    KEY     = 2
    PUBKEYFILE  = "/tmp/user.pub"
    PRIVKEYFILE = "/tmp/user"
)


type testpair struct {
    userprefix string
    password string
    ports []int
}

var data = []testpair {
        {"tr", "1234567890", []int{2001, 2002, 2003, 2004, 2005, 2006, 2007, 2009, 2008, 2010, 2011, 2012, 2013, 2014, 2015, 2016, 2017, 2018, 2019, 2020}},
        {"tr", "0987654321", []int{2001, 2002, 2003, 2004, 2005, 2006, 2007, 2009, 2008, 2010, 2011, 2012, 2013, 2014, 2015, 2016, 2017, 2018, 2019, 2020}},
    }


/**
 * openTunnel
 */
func openTunnel(username string, passwd string, p int, mode int) (*exec.Cmd, error) {    
    port := strconv.Itoa(p)
    var cmdName string
    var cmdArgs []string
    
    if mode == PASSWD {
        cmdName = "sshpass"
        cmdArgs = []string{"-p", passwd, "ssh", "-q", "-t", "-o", "StrictHostkeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-R", "0:localhost:" + port, username + "@localhost", "{\"port\":" + port + "}"}        
    } else {
        cmdName = "ssh"
        cmdArgs = []string{"-i", PRIVKEYFILE, "-q", "-t", "-o", "StrictHostkeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-R", "0:localhost:" + port, username + "@localhost", "{\"port\":" + port + "}"}                
    }
    
    fmt.Println(cmdName, cmdArgs)
	cmd := exec.Command(cmdName, cmdArgs...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    
    err := cmd.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening reverse tunnel", err)
	}

    return cmd, err
}

func establishConns(u *user.User, data testpair, cmd map[int32]*exec.Cmd, mode int, a ConnAddedEvent, r ConnRemovedEvent) {    
    Monitor(u.Uid, a, r)
    
    for _, port := range data.ports {
        cmd[int32(port)], _ = openTunnel(u.Name, data.password, port, mode)
        
        //Rate limit the connections because, too much fork events causes event drops at server.
        //TODO: Need to figure out better way for connection detection.
        time.Sleep(1000 * time.Millisecond)
    }
}

func TestSingleUserConnectionsWithPasswd(t *testing.T) {
    cmd := make(map[int32]*exec.Cmd, len(data[0].ports))
    
    ConnAddEv := func(p int32, h Host) {
        fmt.Printf("Connected %s:%d on Port %d\n", 
                   h.RemoteIP, 
                   h.RemotePort, 
                   h.LocalPort)

        cmd[h.RemotePort].Process.Kill()
        cmd[h.RemotePort].Process.Wait()
    }

    ConnRemoveEv := func(p int32, h Host) {
        fmt.Printf("Removed %s:%d from Port %d\n", 
                   h.RemoteIP, 
                   h.RemotePort, 
                   h.LocalPort)
        delete(cmd, h.RemotePort)
    }
    
    u := user.NewUserWithPassword(data[0].userprefix, data[0].password)

    establishConns(u, data[0], cmd, PASSWD, ConnAddEv, ConnRemoveEv)

    func() {
        for {
            if (len(cmd) == 0) {
                fmt.Println("Finished.")
                break
            }
            runtime.Gosched()
        }   
    }()
    
    u.Delete()
}

func TestMultipleUserConnectionsWithPasswd(t *testing.T) {
    cmd := make([]map[int32]*exec.Cmd, len(data))
    u   := make([]*user.User, len(data))

    for i, _ := range data {
        cmd[i] = make(map[int32]*exec.Cmd, len(data[i].ports))
    
        ConnAddEv := func(p int32, h Host) {
            fmt.Printf("Connected %s:%d on Port %d\n", 
                       h.RemoteIP, 
                       h.RemotePort, 
                       h.LocalPort)

            cmd[i][h.RemotePort].Process.Kill()
            cmd[i][h.RemotePort].Process.Wait()
        }

        ConnRemoveEv := func(p int32, h Host) {
            fmt.Printf("Removed %s:%d from Port %d\n", 
                       h.RemoteIP, 
                       h.RemotePort, 
                       h.LocalPort)
            delete(cmd[i], h.RemotePort)
        }
        u[i] = user.NewUserWithPassword(data[i].userprefix, data[i].password)
        establishConns(u[i], data[i], cmd[i], PASSWD, ConnAddEv, ConnRemoveEv)
    }

    for i, _ := range data {

        func() {
            for {
                if (len(cmd[i]) == 0) {
                    fmt.Println("Finished.")
                    break
                }
                runtime.Gosched()
            }   
        }()

        u[i].Delete()
    }
}

func TestSingleUserConnectionsWithKey(t *testing.T) {
    err := ioutil.WriteFile(PUBKEYFILE, []byte(PUBKEY), 0600)
    if err != nil {
        t.Error("Error writing key file", err)  
    }    

    err = ioutil.WriteFile(PRIVKEYFILE, []byte(PRIVKEY), 0600)
    if err != nil {
        t.Error("Error writing key file", err)  
    }    
    
    cmd := make(map[int32]*exec.Cmd, len(data[0].ports))
    
    ConnAddEv := func(p int32, h Host) {
        fmt.Printf("Connected %s:%d on Port %d\n", 
                   h.RemoteIP, 
                   h.RemotePort, 
                   h.LocalPort)

        cmd[h.RemotePort].Process.Kill()
        cmd[h.RemotePort].Process.Wait()
    }

    ConnRemoveEv := func(p int32, h Host) {
        fmt.Printf("Removed %s:%d from Port %d\n", 
                   h.RemoteIP, 
                   h.RemotePort, 
                   h.LocalPort)
        delete(cmd, h.RemotePort)
    }
    
    u := user.NewUserWithKey(data[0].userprefix, PUBKEYFILE)

    establishConns(u, data[0], cmd, KEY, ConnAddEv, ConnRemoveEv)

    func() {
        for {
            if (len(cmd) == 0) {
                fmt.Println("Finished.")
                break
            }
            runtime.Gosched()
        }   
    }()
    
    u.Delete()    
}