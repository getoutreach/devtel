package store

import "strings"

type IndexMarshaller interface {
	Key() string
	MarshalRecord(addField func(name string, value interface{}))
	UnmarshalRecord(data map[string]interface{}) error
}

func TestAddField(m map[string]interface{}) func(k string, v interface{}) {
	return addField(m)
}

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

func addField(m map[string]interface{}) func(k string, v interface{}) {
	return func(k string, v interface{}) {
		addToMap(m, strings.Split(k, "."), v)
	}
}
