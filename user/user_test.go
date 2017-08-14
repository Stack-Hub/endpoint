/* Copyright (C) Ashish Thakwani - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Ashish Thakwani <athakwani@gmail.com>, August 2017
 */
package user

import (
    "testing"
    "os/exec"
    "fmt"
)

type testpair struct {
    username string
    password string
}

var data = []testpair {
    {"db.3360",    "1234567890"},
    {"app.80",     "0987654321"},
    {"redis.6379", "10293884756"},
    {"elasticsearch.1234", "10293884756"},
    {"logstash.1122", "10293884756"},
    {"mysql.3360", "10293884756"},
    {"postgres.5432", "10293884756"},
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


func Test(t *testing.T) {
    users := make(map[string]*User)
    
    for _, auth := range data {
        u := New(auth.username, auth.password)
        users[auth.username] = u

        err := chkUser(auth.username)
        if err != nil {
            t.Error(
                "For", u.Name,
                "expected", auth.username,
                "got", err,
            )        
        }
    }
    
    for _, u := range users {
        u.Delete()
        err := chkUser(u.Name)
        if err == nil {
            t.Error(
                "User", u.Name,
                "exists",
            )        
        }
    }
}
