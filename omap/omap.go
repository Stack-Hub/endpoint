/* Copyright (C) Ashish Thakwani - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Ashish Thakwani <athakwani@gmail.com>, August 2017
 */
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
    elements    map[int]*Element
    keyList     * list.List
    Userdata    interface{}
}

func New() *OMap {
    var m OMap

    m.nextIdx  = nil
    m.elements = make(map[int]*Element, 1)
    m.keyList  = list.New()
    m.keyList.Init()
    
    return &m
}

func (m *OMap) Add(key int, v interface{}) *Element {
    e := m.keyList.PushBack(key)
    
    m.elements[key] = &Element{
        Value: v,
        keyPtr: e,
    }
    
    return m.elements[key]
}

func (m *OMap) Remove(key int) *Element {
    //Get Element from maps
    e := m.elements[key]
    
    if (e != nil) {
        //Remove key from keylist
        m.keyList.Remove(e.keyPtr)

        //Remove element from map
        delete(m.elements, key)        
    }
    
    return e
}


func (m *OMap) RemoveEl(e *Element) *Element {
    
    if (e != nil) {
        //Get key value
        key := e.keyPtr.Value.(int)
        
        //Remove key from keylist
        m.keyList.Remove(e.keyPtr)

        //Remove element from map
        delete(m.elements, key)        
    }
    
    return e
}


func (m *OMap) Get(key int) *Element {
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
        
    e := m.elements[m.nextIdx.Value.(int)]
    
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
