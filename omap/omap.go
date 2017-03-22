package omap

import (
    "container/list"
)

type Element struct {
    Value interface{}
    keyPtr  * list.Element
}

type OMap struct {
    nextIdx     * list.Element
    elements      map[int32]*Element
    keyList      * list.List
}

func New() *OMap {
    var m OMap

    m.nextIdx  = nil
    m.elements = make(map[int32]*Element, 1)
    m.keyList  = list.New()
    m.keyList.Init()
    
    return &m
}

func (m *OMap) Add(key int32, v interface{}) *Element {
    e := m.keyList.PushBack(key)
    
    m.elements[key] = &Element{
        Value: v,
        keyPtr: e,
    }
    
    return m.elements[key]
}

func (m *OMap) Remove(key int32) *Element {
    //Get Element from maps
    el := m.elements[key]
    
    if (el != nil) {
        //Remove key from keylist
        m.keyList.Remove(el.keyPtr)

        //Remove element from map
        delete(m.elements, key)        
    }
    
    return el
}


func (m *OMap) Get(key int32) *Element {
    //Get Element from maps
    return m.elements[key]

}

func (m *OMap) Next() *Element {
    if len(m.elements) == 0 {
        return nil
    }
    
    if m.nextIdx == nil {
        m.nextIdx = m.keyList.Front()
    }
        
    e := m.elements[m.nextIdx.Value.(int32)]
    
    if m.nextIdx.Next() == nil {
        m.nextIdx = m.keyList.Front()
    } else {
        m.nextIdx = m.nextIdx.Next()
    }
    
    return e    
}

func (m *OMap) Len() int {
    return len(m.elements)
}
