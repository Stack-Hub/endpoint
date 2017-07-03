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
    "io"
    
    "github.com/duppercloud/trafficrouter/utils"
    "github.com/howeyc/fsnotify"
    log "github.com/Sirupsen/logrus"
    "github.com/dchest/safefile"
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
    portmap       format 
    filename      string
    watcher       *fsnotify.Watcher
    event         chan *Event
    isLeader      bool
    joinLeaderGrp bool
    file          *os.File
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
    
    // Clear out old map 
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
    
    if m.joinLeaderGrp {
        // Select Leader based on flag.
        m.sel(data)        
    }
    
    return true
}


/*
 *  Write port map file
 */
func (m * Portmap) write(data *format) bool {

    f, err := safefile.Create(m.filename, 0644)
    if err != nil {
        fmt.Println("Error opening file for writing ", err)
        return false
    } 
    
    defer f.Close()
    
    // encode content
    jsonp, err := json.Marshal(data)
    if err != nil {
        fmt.Println(err)
        return false
    }
    
    log.Debug("portmap = ", string(jsonp))
    
    _, err = io.WriteString(f, string(jsonp))    
    if err != nil {
        fmt.Println("Error writing file ", err)
        return false
    } 

    err = f.Commit()
    if err != nil {
        fmt.Println("Error commiting file ", err)
        return false
    } 
        
    return true
}

/*
 *  If this is first reader then mark it as leader, and disallow others
 */
func (m *Portmap) sel(data *format) {

    // If current instance is already a leader, skip selection process
    if m.isLeader {
        return
    }

    // Set map leader
    m.isLeader = !utils.Itob(data.L)

    // Save to diallow others to become leader
    if m.isLeader {
        // Disallow other to become leader.
        data.L = 1

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

func (m *Portmap) dispatch(ev int, k string, v string) {
    log.Debug("Dispatching ", ev," event for ", k, ":", v)
    m.event <- &Event{Lport: k, Rport: v, Type: ADDED}    
}

/*
 *  Reload file content and generate events to the client.
 */
func (m *Portmap) update(data *format) {
    
    if m.file == nil {        
        
        // For changed entries in temp map generate ADDED event for non leader instance
        for k, v := range data.M {
            log.Debug("Checking ", k, ":", v, " for ADDED event for non leader")
            if old_v, ok := m.portmap.M[k]; ok {
                
                // If this is non leader and value is mapped port then genrate ADDED evnt
                if old_v == "0" && v != "0" && !m.IsLeader() {
                    m.dispatch(ADDED, k, v)
                }                
            }
        }
    
        
        // For new entries in temp map generate ADDED event
        for k, v := range data.M {
            log.Debug("Checking ", k, ":", v, " for ADDED event for leader")
            if _, ok := m.portmap.M[k]; !ok {
                
                // Skip disptaching if rport is 0 and this instance is not a leader.
                if v == "0" && !m.IsLeader() {
                    log.Debug("Skipping v=", v, " m.IsLeader()=", m.IsLeader())
                    continue
                }

                m.dispatch(ADDED, k, v)
            }
        }

        // For missing entries from port map generate DELETED event
        for k, v := range m.portmap.M {
            log.Debug("Checking ", k, ":", v, " for DELETED event")
            if _, ok := data.M[k]; !ok {
                m.dispatch(DELETED, k, v)
            }
        }

        // Update portmap to new loaded map.
        m.portmap = *data
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

                var tempmap format

                // Read file
                ok := m.read(&tempmap)
                if !ok {
                    log.Error("Unable to read map file")
                    return
                }

                m.update(&tempmap)
                
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
func New(name string, joinLeaderGrp bool) (*Portmap, chan *Event) {
    
    m := Portmap{filename: utils.RUNPATH + name, 
                 joinLeaderGrp: joinLeaderGrp}

    // initilize members
    m.event = make(chan *Event, 10)
    
    log.Debug("m.filename=", m.filename)
    
    var tempmap format
    
    // Read file
    ok := m.read(&tempmap)
    if !ok {
        return nil, nil
    }
    
    // Generate events and update map
    m.update(&tempmap)
    
    // Setup file watch
    err := m.watch()
    if err != nil {
        log.Error("Unable to start file watcher", err)
        return nil, nil
    }
    
    return &m, m.event
}

/*
 *  Close and cleanup resources
 */
func (m *Portmap) Close() {
    /* Stop watching */
	m.watcher.Close()    
    
    if m.joinLeaderGrp {
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

    // Only Leader can add port mappings
    if m.isLeader {
        // Add port mapping if no entry exists or entry exists but mapping is "0"
        if mappedPort, ok := m.portmap.M[lport]; !ok || (ok && mappedPort == "0") {

            // Add map and update store file.
            m.portmap.M[lport] = rport
            log.Debug("ADD(): m.portmap", m.portmap)
            m.write(&m.portmap)        

            // Dispatch event only for new entry
            if !ok {
                m.dispatch(ADDED, lport, rport)
            }            
        } 
    }
}
