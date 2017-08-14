/* Copyright (C) Ashish Thakwani - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Ashish Thakwani <athakwani@gmail.com>, August 2017
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