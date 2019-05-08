// Copyright 2017 gf Author(https://github.com/gogf/gf). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with gm file,
// You can obtain one at https://github.com/gogf/gf.
//

package gmap

import (
	"github.com/gogf/gf/g/internal/rwmutex"
	"github.com/gogf/gf/g/util/gconv"
)

type StrIntMap struct {
	mu   *rwmutex.RWMutex
	data map[string]int
}

// NewStrIntMap returns an empty StrIntMap object.
// The param <unsafe> used to specify whether using map with un-concurrent-safety,
// which is false in default, means concurrent-safe.
func NewStrIntMap(unsafe ...bool) *StrIntMap {
	return &StrIntMap{
		mu   : rwmutex.New(unsafe...),
		data : make(map[string]int),
	}
}

// NewStrIntMapFrom returns an StrIntMap object from given map <data>.
// Notice that, the param map is a type of pointer,
// there might be some concurrent-safe issues when changing the map outside.
func NewStrIntMapFrom(data map[string]int, unsafe ...bool) *StrIntMap {
	return &StrIntMap{
		mu   : rwmutex.New(unsafe...),
		data : data,
	}
}

// NewStrIntMapFromArray returns an StrIntMap object from given array.
// The param <keys> is  given as the keys of the map,
// and <values> as its corresponding values.
//
// If length of <keys> is greater than that of <values>,
// the corresponding overflow map values will be the default value of its type.
func NewStrIntMapFromArray(keys []string, values []int, unsafe ...bool) *StrIntMap {
	m := make(map[string]int)
	l := len(values)
	for i, k := range keys {
		if i < l {
			m[k] = values[i]
		} else {
			m[k] = 0
		}
	}
	return &StrIntMap{
		mu   : rwmutex.New(unsafe...),
		data : m,
	}
}

// Iterator iterates the hash map with custom callback function <f>.
// If <f> returns true, then it continues iterating; or false to stop.
func (m *StrIntMap) Iterator(f func(k string, v int) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for k, v := range m.data {
		if !f(k, v) {
			break
		}
	}
}

// Clone returns a new hash map with copy of current map data.
func (m *StrIntMap) Clone() *StrIntMap {
	return NewStrIntMapFrom(m.Map(), !m.mu.IsSafe())
}

// Map returns a copy of the data of the hash map.
func (m *StrIntMap) Map() map[string]int {
	data := make(map[string]int)
	m.mu.RLock()
	for k, v := range m.data {
		data[k] = v
	}
	m.mu.RUnlock()
	return data
}

// Set sets key-value to the hash map.
func (m *StrIntMap) Set(key string, val int) {
	m.mu.Lock()
	m.data[key] = val
	m.mu.Unlock()
}

// Sets batch sets key-values to the hash map.
func (m *StrIntMap) Sets(data map[string]int) {
	m.mu.Lock()
	for k, v := range data {
		m.data[k] = v
	}
	m.mu.Unlock()
}

// Get returns the value by given <key>.
func (m *StrIntMap) Get(key string) int {
	m.mu.RLock()
	val, _ := m.data[key]
	m.mu.RUnlock()
	return val
}

// doSetWithLockCheck checks whether value of the key exists with mutex.Lock,
// if not exists, set value to the map with given <key>,
// or else just return the existing value.
//
// It returns value with given <key>.
func (m *StrIntMap) doSetWithLockCheck(key string, value int) int {
	m.mu.Lock()
	if v, ok := m.data[key]; ok {
		m.mu.Unlock()
		return v
	}
	m.data[key] = value
	m.mu.Unlock()
	return value
}

// GetOrSet returns the value by key,
// or set value with given <value> if not exist and returns this value.
func (m *StrIntMap) GetOrSet(key string, value int) int {
	m.mu.RLock()
	v, ok := m.data[key]
	m.mu.RUnlock()
	if !ok {
		return m.doSetWithLockCheck(key, value)
	} else {
		return v
	}
}

// GetOrSetFunc returns the value by key,
// or sets value with return value of callback function <f> if not exist
// and returns this value.
func (m *StrIntMap) GetOrSetFunc(key string, f func() int) int {
	m.mu.RLock()
	v, ok := m.data[key]
	m.mu.RUnlock()
	if !ok {
		return m.doSetWithLockCheck(key, f())
	} else {
		return v
	}
}

// GetOrSetFuncLock returns the value by key,
// or sets value with return value of callback function <f> if not exist
// and returns this value.
//
// GetOrSetFuncLock differs with GetOrSetFunc function is that it executes function <f>
// with mutex.Lock of the hash map.
func (m *StrIntMap) GetOrSetFuncLock(key string, f func() int) int {
	m.mu.RLock()
	val, ok := m.data[key]
	m.mu.RUnlock()
	if !ok {
		m.mu.Lock()
		defer m.mu.Unlock()
		if v, ok := m.data[key]; ok {
			return v
		}
		val = f()
		m.data[key] = val
		return val
	} else {
		return val
	}
}

// SetIfNotExist sets <value> to the map if the <key> does not exist, then return true.
// It returns false if <key> exists, and <value> would be ignored.
func (m *StrIntMap) SetIfNotExist(key string, value int) bool {
	if !m.Contains(key) {
		m.doSetWithLockCheck(key, value)
		return true
	}
	return false
}

// SetIfNotExistFunc sets value with return value of callback function <f>, then return true.
// It returns false if <key> exists, and <value> would be ignored.
func (m *StrIntMap) SetIfNotExistFunc(key string, f func() int) bool {
	if !m.Contains(key) {
		m.doSetWithLockCheck(key, f())
		return true
	}
	return false
}

// SetIfNotExistFuncLock sets value with return value of callback function <f>, then return true.
// It returns false if <key> exists, and <value> would be ignored.
//
// SetIfNotExistFuncLock differs with SetIfNotExistFunc function is that
// it executes function <f> with mutex.Lock of the hash map.
func (m *StrIntMap) SetIfNotExistFuncLock(key string, f func() int) bool {
	if !m.Contains(key) {
		m.mu.Lock()
		defer m.mu.Unlock()
		if _, ok := m.data[key]; !ok {
			m.data[key] = f()
		}
		return true
	}
	return false
}

// Removes batch deletes values of the map by keys.
func (m *StrIntMap) Removes(keys []string) {
	m.mu.Lock()
	for _, key := range keys {
		delete(m.data, key)
	}
	m.mu.Unlock()
}

// Remove deletes value from map by given <key>, and return this deleted value.
func (m *StrIntMap) Remove(key string) int {
	m.mu.Lock()
	val, exists := m.data[key]
	if exists {
		delete(m.data, key)
	}
	m.mu.Unlock()
	return val
}

// Keys returns all keys of the map as a slice.
func (m *StrIntMap) Keys() []string {
	m.mu.RLock()
	keys := make([]string, 0)
	for key := range m.data {
		keys = append(keys, key)
	}
	m.mu.RUnlock()
	return keys
}

// Values returns all values of the map as a slice.
func (m *StrIntMap) Values() []int {
	m.mu.RLock()
	values := make([]int, 0)
	for _, value := range m.data {
		values = append(values, value)
	}
	m.mu.RUnlock()
	return values
}

// Contains checks whether a key exists.
// It returns true if the <key> exists, or else false.
func (m *StrIntMap) Contains(key string) bool {
	m.mu.RLock()
	_, exists := m.data[key]
	m.mu.RUnlock()
	return exists
}

// Size returns the size of the map.
func (m *StrIntMap) Size() int {
	m.mu.RLock()
	length := len(m.data)
	m.mu.RUnlock()
	return length
}

// IsEmpty checks whether the map is empty.
// It returns true if map is empty, or else false.
func (m *StrIntMap) IsEmpty() bool {
	m.mu.RLock()
	empty := len(m.data) == 0
	m.mu.RUnlock()
	return empty
}

// Clear deletes all data of the map, it will remake a new underlying data map.
func (m *StrIntMap) Clear() {
	m.mu.Lock()
	m.data = make(map[string]int)
	m.mu.Unlock()
}

// LockFunc locks writing with given callback function <f> within RWMutex.Lock.
func (m *StrIntMap) LockFunc(f func(m map[string]int)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f(m.data)
}

// RLockFunc locks reading with given callback function <f> within RWMutex.RLock.
func (m *StrIntMap) RLockFunc(f func(m map[string]int)) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f(m.data)
}

// Flip exchanges key-value of the map, it will change key-value to value-key.
func (m *StrIntMap) Flip() {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := make(map[string]int, len(m.data))
	for k, v := range m.data {
		n[gconv.String(v)] = gconv.Int(k)
	}
	m.data = n
}

// Merge merges two hash maps.
// The <other> map will be merged into the map <m>.
func (m *StrIntMap) Merge(other *StrIntMap) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if other != m {
		other.mu.RLock()
		defer other.mu.RUnlock()
	}
	for k, v := range other.data {
		m.data[k] = v
	}
}