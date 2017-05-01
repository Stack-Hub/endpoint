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
package user

import (
    "testing"
    "io/ioutil"
)

type testpair struct {
    username string
    password string
}

var data = []testpair {
    {"db.3360",    "1234567890"},
    {"app.80",     "0987654321"},
    {"redis.6379", "10293884756"},
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


func TestAdd(t *testing.T) {

    users := make(map[string]*User)
    
    for _, auth := range data {
        u := New(auth.username, auth.password)
        users[username] = u

        err := chkUser(username)
        if err != nil {
            t.Error(
                "For", u.Name,
                "expected", username,
                "got", err,
            )        
        }
    }
    
    for _, u := range users {
        u.Delete()
    }

}
