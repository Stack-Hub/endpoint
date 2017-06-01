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
    "os"
    "strconv"
    "syscall"
    
    "golang.org/x/sys/unix"
    "github.com/duppercloud/trafficrouter/utils"
    "github.com/howeyc/fsnotify"
    log "github.com/Sirupsen/logrus"
)

const (
    ADDED = 0
    DELETED = 1
)

type Event {
    Lport int
    Rport int
    Type  int
}

type format struct {
    m map[string]int `json:"list"`
    l bool           `json:"leader"`
}


/*
 *  Map struct for port mapping 
 */
type Portmap struct {
    portmap  format
    tmpmap   map[string]string
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
    fd := utils.Lockfile(m.filename)
    
    // Reload file
    file, e := ioutil.ReadFile(m.filename)
    if e != nil {
        fmt.Printf("File read error: %v\n", e)
        return nil, nil
    }
	
    // New content in temp map
    json.Unmarshal(file, &m.tempmap)
    
    // For new entries in temp map generate ADDED event
    for k, v := range m.tempmap {
        if m.portmap[k] == nil {
            m.event <- Event{Lport: k, Rport: v, Type: ADDED}
        }
    }

    // For missing entries from port map generate DELETED event
    for k, v := range m.portmap {
        if m.tempmap[k] == nil {
            m.event <- Event{Lport: k, Rport: v, Type: DELETED}
        }
    }
    
    // Update portmap to new loaded map.
    m.portmap = m.tempmap
    delete(m.tempmap)

    // Release file lock
    utils.Unlockfile(fd)
    
}

/*
 *  New port map
 */
func New(name string) (*Portmap, *map[string]string, <- chan *Event) {
    
    m := Portmap{filename: name}

    // initilize members
    m.event = make(chan *Event, 10)
    m.tempmap = make(map[string]string, 1)
    
    // Get exclusive lock on file to avoid corruption.
    fd := utils.Lockfile(m.filename)

    // Read file
    file, e := ioutil.ReadFile(utils.RUNPATH + m.filename)
    if e != nil {
        fmt.Printf("File read error: %v\n", e)
        return nil, nil
    }
    
    // Get watcher for the file events
    m.watcher, err := fsnotify.NewWatcher()
    if err != nil {
        fmt.Printf("File watch error: %v\n", e)
        return nil, nil
	}
    
	// On events reload file and send events to the client.
	go func() {
		for {
			select {
			case ev := <-watcher.Event:
                m.sendEvents()
			case err := <-watcher.Error:
				log.Println("Watcher error:", err, " reloading port map")
                m.sendEvents()
			}
		}
	}()

    // Start watcher
	err = watcher.Watch(utils.RUNPATH + m.filename)
	if err != nil {
		log.Fatal(err)
	}
	
    // Read content
    json.Unmarshal(file, &m.portmap)
    
    if len(m.portmap) == 0 {
        m.portmap = make(map[string]string, 1)
    }    
    
    // Set map leader.
    m.isLeader = !m.portmap.l 
    
    // Disallow other to below leader.
    m.portmap.l = 1
    
    // Save updated map
    m.Save()

    // Release file lock
    utils.Unlockfile(fd)
    
    return &m, &m.portmap, m.Event:
}

/*
 *  Save port map file
 */
func (m * Portmap) Save() {
    jsonp, err := json.Marshal(m.portmap)
    if err != nil {
        fmt.Println(err)
    }
    
    // Get exclusive lock on file to avoid corruption.
    fd := utils.Lockfile(m.filename)

    err = ioutil.WriteFile(utils.RUNPATH + m.filename, jsonp, 0644)
    if err != nil {
        fmt.Println("Error writing file")
    }   
    
    // Release file lock
    utils.Unlockfile(fd)
}

/*
 *  Close and cleanup resources
 */
func (m *Portmap) Close() {
    /* ... do stuff ... */
	m.watcher.Close()    
}

func (m *Portmap) IsLeader() {
    retirn m.isLeader
}