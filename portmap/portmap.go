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
    "github.com/fsnotify/fsnotify"
    log "github.com/Sirupsen/logrus"
)

const (
    ADDED = 0
    UPDATED = 1
    DELETED = 2
)

type Event struct {
    Lport string
    Rport string
    Type  int
}

/*
 *  Map struct for port mapping 
 */
type Portmap struct {
    portmap       map[string]string 
    mapfile       string
    leaderfile    string
    watcher       *fsnotify.Watcher
    event         chan *Event
    isLeader      bool
}

/*
 *  Check if this reader is the leader
 */
func (m *Portmap) IsLeader() bool {
    return m.isLeader
}

/*
 *  Compete for leadership by obtaining exclusive lock on leader file.
 */
func (m *Portmap) electLeader(blocking bool) bool {

    // Retry acquiring lock
    log.Debug("electLeader(): retrying")

    // Check if election is over?
    // Set mode flags to either blocking or non-blocking
    mode := utils.LOCK_EX 
    
    if !blocking {
        mode |= utils.LOCK_NB
    }

    // Acquire lock on leader file
    _, err := utils.LockFile(m.leaderfile, true, mode)
    if err == nil {
        log.Debug("elecLeader(): Leadership Acquired")
        return true
    } 

    return false
        
    /* Don't unlock, because that will make other process leader as well.
     * Only when this process is killed the lock will be released and 
     * other processes will get a chance to become leader.
     */
}

/*
 *  New portmap
 */
func New(name string, electLeader bool) (*Portmap, chan *Event) {
    
    m := Portmap{mapfile: utils.RUNPATH + name + ".map",
                 leaderfile: utils.RUNPATH + name + ".leader"}

    // initilize members
    m.event = make(chan *Event, 10)
    m.portmap = make(map[string]string, 1)
    
    log.Debug("m.mapfile=", m.mapfile)
        
    // Elect leader by acquiring lock on leader file
    // if failed then start async blocking call 
    if electLeader {
        m.isLeader = m.electLeader(false)
        if !m.isLeader {
            go func() {
                m.isLeader = m.electLeader(true)
            }()
        }
    }
    
    // Read file
    ok := m.read()
    if !ok {
        return nil, nil
    }
        
    // Setup file watch
    err := m.watch()
    utils.Check(err)
    
    return &m, m.event
}

/*
 *  Add port mapping
 */
func (m *Portmap) Add(lport string, rport string) {

    log.Debug("ADD(): before m.portmap=", m.portmap)
    ok := m.add(lport, rport)
    log.Debug("ADD(): after m.portmap=", m.portmap)
    
    if ok {
        m.write()
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
            case ev := <-m.watcher.Events:
                log.Println("Recieved watcher event ", ev)
                m.read()
                
			case err := <-m.watcher.Errors:
				log.Println("Watcher error:", err)
			}
		}
	}()

    // Start watcher
	return m.watcher.Add(m.mapfile)
}

/*
 *  Reload file content and generate events to the client.
 */
func (m *Portmap) read() bool {
        
    // Get exclusive lock on file to avoid corruption.
    file, _ := utils.LockFile(m.mapfile, false, utils.LOCK_SH)     

    // Release file lock
    defer utils.UnlockFile(file)
    
    // Read file
    bytes, err := ioutil.ReadAll(file)
    if err != nil {
        fmt.Printf("File read error: %v\n", err)
        return false
    }
    
    var data map[string]string
        
    // Parse content
    err = json.Unmarshal(bytes, &data)
    if err != nil && len(bytes) != 0 {
        log.Error("Error unmarshaling file content", err)
        return false
    }
    
    // Init empty map
    if data == nil {
        log.Debug("read(): Initializing empty map")
        data = make(map [string]string, 1)
    }
    
    log.Debug("Read(): ", string(bytes))
    
    m.update(data)
        
    return true
}

/*
 *  Reload file content and generate events to the client.
 */
func (m *Portmap) update(data map[string]string) {
            
    // For missing entries from port map generate DELETED event
    // DELETE event takes precedance over ADDED because 
    // if the new entry's rport maps to any new entry, then we should remove old entry first.
    for k, v := range m.portmap {
        log.Debug("Checking ", k, ":", v, " for DELETED event")
        if _, ok := data[k]; !ok {
            m.delete(k)
        }
    }

    // Add/update all new entries
    for k, v := range data {
        log.Debug("Adding ", k, ":", v)
        m.add(k, v)
    } 
}

/*
 *  Write port map file
 */
func (m * Portmap) write() bool {

    // Get exclusive lock on file to avoid corruption.
    file, _ := utils.LockFile(m.mapfile, true, utils.LOCK_SH)     

    // Release file lock
    defer utils.UnlockFile(file)
    
    // encode content
    jsonp, err := json.Marshal(m.portmap)
    if err != nil {
        fmt.Println(err)
        return false
    }
    
    log.Debug("portmap = ", string(jsonp))
    
    _, err = file.Write(jsonp)
    if err != nil {
        fmt.Println("Error writing file ", err)
        return false
    } 

    return true
}

func (m *Portmap) dispatch(ev int, k string, v string) {
    log.Debug("Dispatching ", ev," event for ", k, ":", v)
    m.event <- &Event{Lport: k, Rport: v, Type: ev}    
}

/*
 *  Add port entry
 */
func (m *Portmap) add(lport string, rport string) bool {

    // Skip adding incomplete mapping for non leader
    if !m.isLeader && rport == "0" {
        return false
    }
    
    var event = ADDED
    
    if mport, ok := m.portmap[lport]; ok {
        
        // Skip Overwriting if mapping already exists.
        if rport == "0" || rport == mport {
            return false
        }

        event = UPDATED    
    }
    
    m.portmap[lport] = rport
    m.dispatch(event, lport, rport)
    return true
}

/*
 *  Delete
 */
func (m *Portmap) delete(lport string) {
    m.dispatch(DELETED, lport, m.portmap[lport])
    delete(m.portmap, lport)
}

/*
 *  Close and cleanup resources
 */
func (m *Portmap) Close() {
    /* Stop watching */
	m.watcher.Close()    
}

