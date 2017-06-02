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
package portmap

import (
    "fmt"
    "io/ioutil"
    "encoding/json"
    
    "github.com/duppercloud/trafficrouter/utils"
    "github.com/howeyc/fsnotify"
    log "github.com/Sirupsen/logrus"
)

const (
    ADDED = 0
    DELETED = 1
)

type Event struct {
    Lport string
    Rport string
    Type  int
}

type format struct {
    m map[string]string `json:"list"`
    l bool              `json:"leader"`
}


/*
 *  Map struct for port mapping 
 */
type Portmap struct {
    portmap  format
    tempmap  format
    filename string
    watcher  *fsnotify.Watcher
    event    chan *Event
    isLeader bool
}


/*
 *  Reload file content and generate events to the client.
 */
func (m *Portmap) sendEvents() {
    // Get exclusive lock on file to avoid corruption.
    fd := utils.LockFile(utils.RUNPATH + m.filename)
    
    // Reload file
    file, e := ioutil.ReadFile(utils.RUNPATH + m.filename)
    if e != nil {
        fmt.Printf("File read error: %v\n", e)
        return
    }
	
    // New content in temp map
    json.Unmarshal(file, &m.tempmap)
    
    // For new entries in temp map generate ADDED event
    for k, v := range m.tempmap.m {
        if _, ok := m.portmap.m[k]; !ok {
            m.event <- &Event{Lport: k, Rport: v, Type: ADDED}
        }
    }

    // For missing entries from port map generate DELETED event
    for k, v := range m.portmap.m {
        if _, ok := m.tempmap.m[k]; !ok {
            m.event <- &Event{Lport: k, Rport: v, Type: DELETED}
        }
    }
    
    // Update portmap to new loaded map.
    m.portmap = m.tempmap

    // Release file lock
    utils.UnlockFile(fd)
    
}

/*
 *  New port map
 */
func New(name string) (*Portmap, map[string]string, <- chan *Event) {
    
    m := Portmap{filename: name}

    // initilize members
    m.event = make(chan *Event, 10)
    m.tempmap.m = make(map[string]string, 1)
    
    // Get exclusive lock on file to avoid corruption.
    fd := utils.LockFile(utils.RUNPATH + m.filename)

    // Read file
    file, e := ioutil.ReadFile(utils.RUNPATH + m.filename)
    if e != nil {
        fmt.Printf("File read error: %v\n", e)
        return nil, nil, nil
    }
    
    // Get watcher for the file events
    var err error
    m.watcher, err = fsnotify.NewWatcher()
    if err != nil {
        fmt.Printf("File watch error: %v\n", e)
        return nil, nil, nil
	}
    
	// On events reload file and send events to the client.
	go func() {
		for {
			select {
			case  <-m.watcher.Event:
                m.sendEvents()
			case err := <-m.watcher.Error:
				log.Println("Watcher error:", err)
			}
		}
	}()

    // Start watcher
	err = m.watcher.Watch(utils.RUNPATH + m.filename)
	if err != nil {
		log.Fatal(err)
	}
	
    // Read content
    json.Unmarshal(file, &m.portmap)
    
    if len(m.portmap.m) == 0 {
        m.portmap.m = make(map[string]string, 1)
    }    
    
    // Set map leader.
    m.isLeader = !m.portmap.l 
    
    // Disallow other to below leader.
    m.portmap.l = true
    
    // Save updated map
    m.save()

    // Release file lock
    utils.UnlockFile(fd)
    
    return &m, m.portmap.m, m.event
}

/*
 *  Save port map file
 */
func (m * Portmap) save() {
    jsonp, err := json.Marshal(m.portmap)
    if err != nil {
        fmt.Println(err)
    }
    
    err = ioutil.WriteFile(utils.RUNPATH + m.filename, jsonp, 0644)
    if err != nil {
        fmt.Println("Error writing file")
    }   
}

/*
 *  Close and cleanup resources
 */
func (m *Portmap) Close() {
    /* ... do stuff ... */
	m.watcher.Close()    
}

/*
 *  Check if this reader is the leader
 */
func (m *Portmap) IsLeader() bool {
    return m.isLeader
}

/*
 *  Add port mapping
 */
func (m *Portmap) Add(lport string, rport string) {
    
    m.sendEvents()
    m.portmap.m[lport] = rport

    // Get exclusive lock on file to avoid corruption.
    fd := utils.LockFile(utils.RUNPATH + m.filename)
    
    m.save()
    
    // Release file lock
    utils.UnlockFile(fd)
    
}
