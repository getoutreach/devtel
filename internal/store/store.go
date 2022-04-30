package store

import (
	"encoding/json"
	"io"
	"sort"
)

type entry struct {
	Key       string      `json:"key"`
	Data      interface{} `json:"data"`
	Processed bool        `json:"processed,omitempty"`
}

type Store interface {
	Append(value interface{}) error

	Get(key string) interface{}
	GetAll() []interface{}
	GetUnprocessed() []interface{}

	MarkProcessed([]interface{}) error
	Restore(r io.Reader, caster func(map[string]interface{}) interface{}) error
}

type store struct {
	w       io.Writer
	encoder *json.Encoder
	entries []entry
	index   map[string]int

	key func(interface{}) string
}

func NewWithWriter(key func(interface{}) string, w io.Writer) Store {
	return &store{
		key:     key,
		w:       w,
		encoder: json.NewEncoder(w),
	}
}

func (s *store) Append(value interface{}) error {
	return s.append(value, false)
}

func (s *store) append(value interface{}, processed bool) error {
	e := entry{s.key(value), value, processed}
	if err := s.encoder.Encode(e); err != nil {
		return err
	}
	s.appendEntry(e)

	return nil
}

func (s *store) appendEntry(e entry) {
	if s.index == nil {
		s.index = make(map[string]int)
	}
	s.index[e.Key] = len(s.entries)
	s.entries = append(s.entries, e)
}

func (s *store) Get(key string) interface{} {
	if i, ok := s.index[key]; ok {
		return s.entries[i].Data
	}
	return nil
}

func (s *store) GetAll() []interface{} {
	indexes := make([]int, 0, len(s.entries))
	for _, index := range s.index {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	var values []interface{}
	for _, index := range indexes {
		values = append(values, s.entries[index].Data)
	}

	return values
}

func (s *store) GetUnprocessed() []interface{} {
	indexes := make([]int, 0, len(s.entries))
	for _, index := range s.index {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	var values []interface{}
	for _, index := range indexes {
		val := s.entries[index]
		if !val.Processed {
			values = append(values, val.Data)
		}
	}

	return values
}

func (s *store) MarkProcessed(recs []interface{}) error {
	for _, rec := range recs {
		if err := s.append(rec, true); err != nil {
			return err
		}
	}

	return nil
}

func (s *store) Restore(r io.Reader, caster func(val map[string]interface{}) interface{}) error {
	dec := json.NewDecoder(r)

	for dec.More() {
		var e entry
		if err := dec.Decode(&e); err != nil {
			return err
		}
		// JSON transforms into map[string]interface{}
		e.Data = caster(e.Data.(map[string]interface{}))

		s.appendEntry(e)
	}

	return nil
}