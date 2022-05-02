// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains code related to encoding and decoding events.

package store

import "strings"

// IndexMarshaller is the interface for marshalling and unmarshalling index data.
type IndexMarshaller interface {
	Key() string
	MarshalRecord(addField func(name string, value interface{}))
	UnmarshalRecord(data map[string]interface{}) error
}

// addToMap adds a value to a map based on a path. If the path is not found, new maps will be added.
func addToMap(m map[string]interface{}, path []string, v interface{}) {
	if len(path) == 0 {
		return
	}

	if len(path) == 1 {
		m[path[0]] = v
		return
	}

	if _, ok := m[path[0]]; !ok {
		m[path[0]] = make(map[string]interface{})
	}

	if _, ok := m[path[0]].(map[string]interface{}); !ok {
		m[path[0]] = make(map[string]interface{})
	}

	addToMap(m[path[0]].(map[string]interface{}), path[1:], v)
}

// addField creates a function that adds a field to a map.
func addField(m map[string]interface{}) func(k string, v interface{}) {
	return func(k string, v interface{}) {
		addToMap(m, strings.Split(k, "."), v)
	}
}
