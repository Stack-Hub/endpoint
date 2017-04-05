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
    elements      map[int]*Element
    keyList      * list.List
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
