/* Copyright (C) Ashish Thakwani - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Ashish Thakwani <athakwani@gmail.com>, August 2017
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
    watcher       *fsnotify.Watcher
    event         chan *Event
}

var portmaps []Portmap

func Cleanup() {
    for _, m := range portmaps {
        m.Close()
    }
}

/*
 *  New portmap
 */
func New(name string) (*Portmap, chan *Event) {
    
    m := Portmap{mapfile: utils.RUNPATH + name + ".map"}

    portmaps = append(portmaps, m)
    
    // initilize members
    m.event = make(chan *Event, 10)
    m.portmap = make(map[string]string, 1)
    
    //log.Debug("m.mapfile=", m.mapfile)
            
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

    //log.Debug("ADD(): before m.portmap=", m.portmap)
    m.add(lport, rport)
    //log.Debug("ADD(): after m.portmap=", m.portmap)
    m.write(lport, rport, false)
}

/*
 *  Add port mapping
 */
func (m *Portmap) Delete(lport string) {

    m.delete(lport)    
    m.write(lport, "0", true)
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
            case <-m.watcher.Events:
                //log.Debug("Recieved watcher event ", ev)
                m.read()
                
			case err := <-m.watcher.Errors:
				log.Debug("Watcher error:", err)
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
        //log.Debug("read(): Initializing empty map")
        data = make(map [string]string, 1)
    }

    //log.Debug("Read(): ", string(bytes))

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
    for k, _ := range m.portmap {
        //log.Debug("Checking ", k, ":", v, " for DELETED event")
        if _, ok := data[k]; !ok {
            m.delete(k)
        }
    }

    // Add/update all new entries
    for k, v := range data {
        //log.Debug("Adding ", k, ":", v)
        m.add(k, v)
    } 
}

/*
 *  Write port map file
 */
func (m * Portmap) write(lport string, rport string, isDelete bool) bool {

    // Get exclusive lock on file to avoid corruption.
    file, _ := utils.LockFile(m.mapfile, false, utils.LOCK_SH)     

    // Release file lock
    defer utils.UnlockFile(file)

    // Read file
    bytes, err := ioutil.ReadAll(file)
    if err != nil {
        fmt.Printf("write(): File read error: %v\n", err)
        return false
    }

    var data map[string]string

    // Parse content
    err = json.Unmarshal(bytes, &data)
    if err != nil && len(bytes) != 0 {
        log.Error("write(): Error unmarshaling file content", err)
        return false
    }

    // Init empty map
    if data == nil {
        log.Debug("write(): Initializing empty map")
        data = make(map [string]string, 1)
    }    
    
    if !isDelete {
        if _, ok := data[lport]; !ok {
            // Add port mapping
            data[lport] = rport            
        }
    } else {
        delete(data, lport)
    }
    
    // encode content
    jsonp, err := json.Marshal(data)
    if err != nil {
        fmt.Println(err)
        return false
    }

    //log.Debug("write(): portmap = ", string(jsonp))

    //Empty file first
    file.Truncate(0)
    file.Seek(0,0)
    
    _, err = file.Write(jsonp)
    if err != nil {
        fmt.Println("write(): Error writing file ", err)
        return false
    }         

    return true
}

func (m *Portmap) dispatch(ev int, k string, v string) {
    //log.Debug("Dispatching ", ev," event for ", k, ":", v)
    m.event <- &Event{Lport: k, Rport: v, Type: ev}    
}

/*
 *  Add port entry
 */
func (m *Portmap) add(lport string, rport string) {
    var event = ADDED
    
    if _, ok := m.portmap[lport]; ok {
        event = UPDATED    
    }
    
    m.portmap[lport] = rport
    m.dispatch(event, lport, rport)
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
    utils.DeleteFile(m.mapfile)
}

