// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This package contains the implementation of append-only index of telemetry events.

// Package store contains the implementation of Store.
// Store provides a simple append-only index of telemetry events. On append it marshals the data into JSON
// and appends it to the log file. The events are managed based on a key. Key is provided by the caller.
// It also tracks whether the event has been processed or not. This is useful for determining if the event
// needs to be sent to telemetry or not.
package store

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/getoutreach/gobox/pkg/trace"
	"github.com/pkg/errors"
)

// entry is index entry for wrapping the event, tracking event key, and whether the event was processed.
type entry struct {
	Key       string                 `json:"key"`
	Data      map[string]interface{} `json:"data"`
	Processed bool                   `json:"processed,omitempty"`
}

// bag is an map alias that provides a MarshalRecord method. It's used to hold default fields.
type bag map[string]interface{}

func (b bag) MarshalRecord(addField func(name string, value interface{})) {
	for k, v := range b {
		addField(k, v)
	}
}

// Store provides a generic append-only index of data.
type Store interface {
	Init(context.Context) error

	AddDefaultField(string, interface{})

	Append(context.Context, IndexMarshaller) error

	Get(context.Context, string, IndexMarshaller) error
	GetAll(context.Context) *Cursor
	GetUnprocessed(context.Context) *Cursor

	MarkProcessed(context.Context, []IndexMarshaller) error
}

// FSStore is the concrete implementation of Store.
type FSStore struct {
	logDir     string
	logPath    string
	logFS      fs.FS
	openAppend func(path string) (io.WriteCloser, error)

	entries       []entry
	index         map[string]int
	defaultFields bag
}

// Options hold the store configuration.
type Options struct {
	LogDir     string
	LogFS      fs.FS
	OpenAppend func(path string) (io.WriteCloser, error)
}

// New creates a new FSStore instance.
func New(opts *Options) *FSStore {
	if opts.LogDir == "" {
		opts.LogDir = filepath.Join(os.TempDir(), "devtel")
	}

	if opts.LogFS == nil {
		opts.LogFS = os.DirFS(opts.LogDir)
	}

	if opts.OpenAppend == nil {
		opts.OpenAppend = func(path string) (io.WriteCloser, error) {
			return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		}
	}

	return &FSStore{
		logDir:        opts.LogDir,
		logFS:         opts.LogFS,
		openAppend:    opts.OpenAppend,
		defaultFields: bag{},
	}
}

// Init goes through all the log files in the log dir and loads the entries into the store.
// It either creates a log file, or uses the last one if it exists.
func (s *FSStore) Init(ctx context.Context) error {
	ctx = trace.StartCall(ctx, "store.Init")
	defer trace.EndCall(ctx)

	if s.logDir != "" {
		if err := os.MkdirAll(s.logDir, 0o755); err != nil {
			return trace.SetCallStatus(ctx, err)
		}
	}

	err := fs.WalkDir(s.logFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		f, err := s.logFS.Open(path)
		if err != nil {
			return errors.Wrapf(err, "failed to open %s", path)
		}
		defer f.Close()

		if err := s.restore(f); err != nil {
			return errors.Wrapf(err, "failed to restore %s", path)
		}

		s.logPath = filepath.Join(s.logDir, path)

		return nil
	})

	if err != nil {
		return trace.SetCallStatus(ctx, err)
	}

	if s.logPath == "" {
		s.logPath = filepath.Join(s.logDir, fmt.Sprintf("%d.log", time.Now().Unix()))
	}

	return trace.SetCallStatus(ctx, err)
}

// AddDefaultField adds a default field to the store. These fields are added to all events.
func (s *FSStore) AddDefaultField(k string, v interface{}) {
	s.defaultFields[k] = v
}

// Append adds an event to the store.
func (s *FSStore) Append(ctx context.Context, value IndexMarshaller) error {
	ctx = trace.StartCall(ctx, "store.Append")
	defer trace.EndCall(ctx)

	return trace.SetCallStatus(ctx, s.append(value, false))
}

// append adds an event to the store.
// It adds default fields, and marshals the data. Then it appends the to the log file and in-memory index.
func (s *FSStore) append(value IndexMarshaller, processed bool) error {
	val := make(map[string]interface{})

	adder := addField(val)
	s.defaultFields.MarshalRecord(adder)
	value.MarshalRecord(adder)

	e := entry{value.Key(), val, processed}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	f, err := s.openAppend(s.logPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintln(f, string(b))
	if err != nil {
		return err
	}

	s.appendEntry(e)

	return nil
}

// appendEntry adds an entry to the in-memory index.
func (s *FSStore) appendEntry(e entry) {
	if s.index == nil {
		s.index = make(map[string]int)
	}
	s.index[e.Key] = len(s.entries)
	s.entries = append(s.entries, e)
}

// Get gets an event from the store.
func (s *FSStore) Get(ctx context.Context, key string, value IndexMarshaller) error {
	ctx = trace.StartCall(ctx, "store.Get")
	defer trace.EndCall(ctx)

	if i, ok := s.index[key]; ok {
		return trace.SetCallStatus(ctx, value.UnmarshalRecord(s.entries[i].Data))
	}

	return nil
}

// GetAll returns all the events in the store. The latest versions of the events.
func (s *FSStore) GetAll(ctx context.Context) *Cursor {
	ctx = trace.StartCall(ctx, "store.GetAll")
	defer trace.EndCall(ctx)

	indexes := make([]int, 0, len(s.entries))
	for _, index := range s.index {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	var values []map[string]interface{}
	for _, index := range indexes {
		values = append(values, s.entries[index].Data)
	}

	return NewCursor(values)
}

// GetUnprocessed returns all the events in the store that have not been processed.
func (s *FSStore) GetUnprocessed(ctx context.Context) *Cursor {
	ctx = trace.StartCall(ctx, "store.GetUnprocessed")
	defer trace.EndCall(ctx)

	indexes := make([]int, 0, len(s.entries))
	for _, index := range s.index {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	var values []map[string]interface{}
	for _, index := range indexes {
		val := s.entries[index]
		if !val.Processed {
			values = append(values, val.Data)
		}
	}

	return NewCursor(values)
}

// MarkProcessed marks the events as processed.
func (s *FSStore) MarkProcessed(ctx context.Context, recs []IndexMarshaller) error {
	ctx = trace.StartCall(ctx, "store.MarkProcessed")
	defer trace.EndCall(ctx)

	for _, rec := range recs {
		if err := s.append(rec, true); err != nil {
			return trace.SetCallStatus(ctx, err)
		}
	}

	return nil
}

// restore reads the log file and adds the entries to the in-memory index.
func (s *FSStore) restore(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var e entry

		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			return errors.Wrap(err, "failed to unmarshal entry")
		}
		if e.Key == "" {
			continue
		}
		if e.Data == nil {
			continue
		}

		s.appendEntry(e)
	}

	return nil
}
