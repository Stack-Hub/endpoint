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
    "os"
    
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
    M map[string]string `json:"list"`
    L int               `json:"leader"`
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
    file     *os.File
}

func itob(b int) bool {
    if b > 0 {
        return true
    }
    return false
 }

/*
 *  Reload file content and generate events to the client.
 */
func (m *Portmap) read(data *format) bool {
    // Get exclusive lock on file to avoid corruption.
    m.file = utils.LockFile(m.filename)     

    // Read file
    bytes, e := ioutil.ReadAll(m.file)
    if e != nil {
        fmt.Printf("File read error: %v\n", e)
        return false
    }
    	
    // Parse content
    json.Unmarshal(bytes, data)
    
    // Select Leader based on file flag.
    m.leaderSelect()
    
    // Release file lock
    utils.UnlockFile(m.file)
    m.file = nil
    return true
}


/*
 *  Write port map file
 */
func (m * Portmap) write() bool {
    
    isLocked := false
    
    if m.file == nil {
        // Get exclusive lock on file to avoid corruption.
        m.file = utils.LockFile(m.filename)             
        isLocked = true
    }

    // encode content
    jsonp, err := json.Marshal(&m.portmap)
    if err != nil {
        fmt.Println(err)
        return false
    }
    
    log.Debug("portmap = ", string(jsonp))
    
    //TODO: Add write count check
    _, err = m.file.Write(jsonp)
    if err != nil {
        fmt.Println("Error writing file")
        return false
    }   
    
    if isLocked {
        // Release file lock
        utils.UnlockFile(m.file) 
        m.file = nil
    }

    return true
}

/*
 *  If this is first reader then mark it as leader, and disallow others
 */
func (m *Portmap) leaderSelect() {
    // Set map leader
    m.isLeader = !itob(m.portmap.L)

    // Save to diallow others to become leader
    if m.isLeader {
        // Disallow other to become leader.
        m.portmap.L = 1

        // Save updated map
        m.write()
    }
}

/*
 *  Reload file content and generate events to the client.
 */
func (m *Portmap) leaderDeselect() {
    // Save to diallow others to become leader
    if m.isLeader {
        // Disallow other to become leader.
        m.portmap.L = 0

        // Save updated map
        m.write()
    }
}

/*
 *  Reload file content and generate events to the client.
 */
func (m *Portmap) update() {

    if m.file != nil {        
        
        // Reload file
        ok := m.read(&m.tempmap)
        if !ok {
            return
        }

        // For new entries in temp map generate ADDED event
        for k, v := range m.tempmap.M {
            if _, ok := m.portmap.M[k]; !ok {
                m.event <- &Event{Lport: k, Rport: v, Type: ADDED}
            }
        }

        // For missing entries from port map generate DELETED event
        for k, v := range m.portmap.M {
            if _, ok := m.tempmap.M[k]; !ok {
                m.event <- &Event{Lport: k, Rport: v, Type: DELETED}
            }
        }

        // Update portmap to new loaded map.
        m.portmap = m.tempmap
    }    
}

/*
 *  New port map
 */
func New(name string) (*Portmap, map[string]string, chan *Event) {
    
    m := Portmap{filename: utils.RUNPATH + name}

    // initilize members
    m.event = make(chan *Event, 10)
    m.tempmap.M = make(map[string]string, 1)
    
    log.Debug("m.filename=", m.filename)
    
    // Reload file
    ok := m.read(&m.portmap)
    if !ok {
        return nil, nil, nil
    }
        
    if len(m.portmap.M) == 0 {
        m.portmap.M = make(map[string]string, 1)
    }    
        
    // Get watcher for the file events
    var err error
    m.watcher, err = fsnotify.NewWatcher()
    if err != nil {
        fmt.Printf("File watch error: %v\n", err)
        return nil, nil, nil
	}
    
	// On events reload file and send events to the client.
	go func() {
		for {
			select {
            case ev := <-m.watcher.Event:
                log.Println("Recieved watcher event ", ev)
                m.update()
			case err := <-m.watcher.Error:
				log.Println("Watcher error:", err)
			}
		}
	}()

    // Start watcher
	err = m.watcher.WatchFlags(m.filename, fsnotify.FSN_CREATE | fsnotify.FSN_MODIFY | fsnotify.FSN_DELETE )
	if err != nil {
		log.Fatal(err)
	}
    
    return &m, m.portmap.M, m.event
}

/*
 *  Close and cleanup resources
 */
func (m *Portmap) Close() {
    /* Stop watching */
	m.watcher.Close()    
    
    m.leaderDeselect()
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
    m.portmap.M[lport] = rport
    log.Debug("m.portmap", m.portmap)
    m.write()    
}
