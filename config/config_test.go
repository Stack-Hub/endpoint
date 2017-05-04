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
package config

import (
    "testing"
    "fmt"
    "os"
    "strconv"
    "encoding/json"
    "net"
    
    "github.com/duppercloud/trafficrouter/utils"
)

type testpair struct {
    ListenPort  uint32 `json:"lisport"` // Localhost listening port of reverse tunnel
    RemoteIP    string `json:"raddr"`   // Remote IP 
    RemotePort  uint32 `json:"rport"`   // Port on which remote host connected
}

var data = []testpair{
    {1000, "192.168.1.1", 2345}
}

func TestSend(t *testing.T){
    
}