package omap

import (
    "testing"
)

type Config struct {
    Port uint32 `json:"port"`
    Mode string `json:"mode"`
}

type Host struct {
    localPort  int32
    remoteIP   string
    remotePort int32
    config     Config
}

type testpair struct {
  key int32
  value Host
}


var data = []testpair {
                        {1, Host{ 1000, "192.168.1.1", 1000, Config {1000, "detach"}}},
                        {2, Host{ 2000, "192.168.1.2", 2000, Config {2000, "detach"}}},
                        {3, Host{ 3000, "192.168.1.2", 3000, Config {3000, "detach"}}},
                        {4, Host{ 4000, "192.168.1.2", 4000, Config {4000, "detach"}}},
                        {5, Host{ 5000, "192.168.1.2", 5000, Config {5000, "detach"}}},
                      }

func initData() (* OMap) {
    m := New()
    
    for _, conn := range data {
            m.Add(conn.key, conn.value)
    }    
    
    return m
}

func TestEmpty(t *testing.T) {
    m := New()
    
    el := m.Next();
    t.Log("Element =", el)
    if el != nil {
        t.Error(
            "For", m,
            "expected", nil,
            "got", el,
        )
    }
}


func TestNext(t *testing.T) {    
    
    //Initialize Data in OrderedMap
    m := initData()
    
    for _, conn := range data {
        el := m.Next()
        t.Log(conn.value, el.Value.(Host))
        if el.Value.(Host).localPort != conn.value.localPort {
            t.Error(
                "For", conn.value,
                "expected", conn.value.localPort,
                "got", *el,
            )
        }
    }
}

func TestNextLoop(t *testing.T) {
    //Initialize Data in OrderedMap
    m := initData()
    
    for _, conn := range data {
        el := m.Next()
        t.Log(conn.value, el.Value.(Host))
        if el.Value.(Host).localPort != conn.value.localPort {
            t.Error(
                "For", conn.value,
                "expected", conn.value.localPort,
                "got", *el,
            )
        }
    }

    for _, conn := range data {
        el := m.Next()
        t.Log(conn.value, el.Value.(Host))
        if el.Value.(Host).localPort != conn.value.localPort {
            t.Error(
                "For", conn.value,
                "expected", conn.value.localPort,
                "got", *el,
            )
        }
    }

}

func TestRemove(t *testing.T) {
    const removeIdx = 2
    
    //Initialize Data in OrderedMap
    m := initData()
    
    m.Remove(data[removeIdx].key)
    
    for idx, conn := range data {
        if (idx == removeIdx) {
            continue            
        }

        el := m.Next()
        t.Log(conn.value, el.Value.(Host))
        
        if el.Value.(Host).localPort != conn.value.localPort {
            t.Error(
                "For", conn.value,
                "expected", conn.value.localPort,
                "got", *el,
            )
        }
    }
}

func TestRemoveLoop(t *testing.T) {
    const removeIdx = 2
    
    //Initialize Data in OrderedMap
    m := initData()
    
    m.Remove(data[removeIdx].key)
    
    for idx, conn := range data {
        if (idx == removeIdx) {
            continue            
        }

        el := m.Next()
        t.Log(conn.value, el.Value.(Host))
        
        if el.Value.(Host).localPort != conn.value.localPort {
            t.Error(
                "For", conn.value,
                "expected", conn.value.localPort,
                "got", *el,
            )
        }
    }

    for idx, conn := range data {
        if (idx == removeIdx) {
            continue            
        }

        el := m.Next()
        t.Log(conn.value, el.Value.(Host))
        
        if el.Value.(Host).localPort != conn.value.localPort {
            t.Error(
                "For", conn.value,
                "expected", conn.value.localPort,
                "got", *el,
            )
        }
    }


}

func TestGet(t *testing.T) {
    
    //Initialize Data in OrderedMap
    m := initData()
    
    for _, conn := range data {
        el := m.Get(conn.key)
        t.Log(conn.value, el.Value.(Host))
        if el.Value.(Host).localPort != conn.value.localPort {
            t.Error(
                "For", conn.value,
                "expected", conn.value.localPort,
                "got", *el,
            )
        }
    }
    
}
