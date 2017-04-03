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
package server

import (
    "testing"
    "strconv"
    "fmt"
    "os"
    "os/exec"
    "io/ioutil"
    "runtime"
    "time"
    
    "../user"
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


func TestSingleUserConnectionsWithPasswd(t *testing.T) {
    cmd := make(map[int32]*exec.Cmd, len(data[0].ports))
    
    ConnAddEv := func(p int32, h Host) {
        fmt.Printf("Connected %s:%d on Port %d\n", 
                   h.RemoteIP, 
                   h.RemotePort, 
                   h.LocalPort)
    }

    ConnRemoveEv := func(p int32, h Host) {
        fmt.Printf("Removed %s:%d from Port %d\n", 
                   h.RemoteIP, 
                   h.RemotePort, 
                   h.LocalPort)
        delete(cmd, h.RemotePort)
    }
    
    Monitor(a, r)

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
