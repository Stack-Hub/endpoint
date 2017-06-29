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
    portmap   format 
    filename  string
    watcher   *fsnotify.Watcher
    event     chan *Event
    isLeader  bool
    chkLeader bool
    file      *os.File
}

/*
 *  Reload file content and generate events to the client.
 */
func (m *Portmap) read(data *format) bool {
    // Get exclusive lock on file to avoid corruption.
    m.file = utils.LockFile(m.filename, false)     

    // Release file lock
    defer func() {
        utils.UnlockFile(m.file)
        m.file = nil
    }()
    
    // Read file
    bytes, err := ioutil.ReadAll(m.file)
    if err != nil {
        fmt.Printf("File read error: %v\n", err)
        return false
    }
    
    // Clear out old map by creating new one
    data.M = nil
    
    // Parse content
    err = json.Unmarshal(bytes, data)
    if err != nil && len(bytes) != 0 {
        log.Error("Error unmarshaling file content", err)
        return false
    }
    
    // Init empty map
    if data.M == nil {
        data.M = make(map [string]string, 1)
    }
    
    log.Debug("Read: ", string(bytes))
    
    if m.chkLeader {
        // Select Leader based on flag.
        m.sel(data)        
    }
    
    return true
}


/*
 *  Write port map file
 */
func (m * Portmap) write(data *format) bool {
        
    if m.file == nil {
        // Get exclusive lock on file to avoid corruption.
        m.file = utils.LockFile(m.filename, true)             

        defer func() {
            // Release file lock
            utils.UnlockFile(m.file) 
            m.file = nil
        }()
    }

    // encode content
    jsonp, err := json.Marshal(data)
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
    
    return true
}

/*
 *  If this is first reader then mark it as leader, and disallow others
 */
func (m *Portmap) sel(data *format) {
    // Set map leader
    m.isLeader = !utils.Itob(data.L)

    // Save to diallow others to become leader
    if m.isLeader {
        // Disallow other to become leader.
        m.portmap.L = 1

        // Save updated map
        m.write(data)
    }
}

/*
 *  Reload file content and generate events to the client.
 */
func (m *Portmap) desel(data *format) {
    // Save to diallow others to become leader
    if m.isLeader {
        // Allow other to become leader.
        data.L = 0

        // Save updated map
        m.write(data)
    }
}

/*
 *  Reload file content and generate events to the client.
 */
func (m *Portmap) update() {

    var tempmap format 
    
    if m.file == nil {        
        
        // Reload file
        ok := m.read(&tempmap)
        if !ok {
            return
        }

        // For new entries in temp map generate ADDED event
        for k, v := range tempmap.M {
            log.Debug("Checking ", k, ":", v, "for ADDED event")
            if _, ok := m.portmap.M[k]; !ok {
                log.Debug("Dispatching ADDED event for ", k, ":", v)
                m.event <- &Event{Lport: k, Rport: v, Type: ADDED}
            }
        }

        // For missing entries from port map generate DELETED event
        for k, v := range m.portmap.M {
            log.Debug("Checking ", k, ":", v, "for DELETED event")
            if _, ok := tempmap.M[k]; !ok {
                log.Debug("Dispatching DELETED event for ", k, ":", v)
                m.event <- &Event{Lport: k, Rport: v, Type: DELETED}
            }
        }

        // Update portmap to new loaded map.
        m.portmap = tempmap
    }    
}

/*
 *  Implements fsnotify file system watcher
 */
func (m *Portmap) watch() error {

    // Get watcher for the file events
    var err error
    m.watcher, err = fsnotify.NewWatcher()
    if err != nil {
        fmt.Printf("File watch error: %v\n", err)
        return err
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
	return m.watcher.WatchFlags(m.filename, fsnotify.FSN_CREATE | fsnotify.FSN_MODIFY | fsnotify.FSN_DELETE)
}

/*
 *  New portmap
 */
func New(name string, chkleader bool) (*Portmap, map[string]string, chan *Event) {
    
    m := Portmap{filename: utils.RUNPATH + name, 
                 chkLeader: chkleader}

    // initilize members
    m.event = make(chan *Event, 10)
    
    log.Debug("m.filename=", m.filename)
    
    // Read file
    ok := m.read(&m.portmap)
    if !ok {
        return nil, nil, nil
    }
    
    // Setup file watch
    err := m.watch()
    if err != nil {
        log.Error("Unable to start file watcher", err)
        return nil, nil, nil
    }
    
    return &m, m.portmap.M, m.event
}

/*
 *  Close and cleanup resources
 */
func (m *Portmap) Close() {
    /* Stop watching */
	m.watcher.Close()    
    
    if m.chkLeader {
        m.desel(&m.portmap)
    }
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
    m.write(&m.portmap)    
}
