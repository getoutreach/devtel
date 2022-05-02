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

type entry struct {
	Key       string                 `json:"key"`
	Data      map[string]interface{} `json:"data"`
	Processed bool                   `json:"processed,omitempty"`
}

type bag map[string]interface{}

func (b bag) MarshalRecord(addField func(name string, value interface{})) {
	for k, v := range b {
		addField(k, v)
	}
}

type Store interface {
	Init(context.Context) error

	AddDefaultField(string, interface{})

	Append(context.Context, IndexMarshaller) error

	Get(context.Context, string, IndexMarshaller) error
	GetAll(context.Context) *Cursor
	GetUnprocessed(context.Context) *Cursor

	MarkProcessed(context.Context, []IndexMarshaller) error
}

type store struct {
	logDir     string
	logPath    string
	logFS      fs.FS
	openAppend func(path string) (io.WriteCloser, error)

	entries       []entry
	index         map[string]int
	defaultFields bag
}

type Options struct {
	LogDir     string
	LogFS      fs.FS
	OpenAppend func(path string) (io.WriteCloser, error)
}

func New(opts *Options) Store {
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

	return &store{
		logDir:        opts.LogDir,
		logFS:         opts.LogFS,
		openAppend:    opts.OpenAppend,
		defaultFields: bag{},
	}
}

func (s *store) Init(ctx context.Context) error {
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

func (s *store) AddDefaultField(k string, v interface{}) {
	s.defaultFields[k] = v
}

func (s *store) Append(ctx context.Context, value IndexMarshaller) error {
	ctx = trace.StartCall(ctx, "store.Append")
	defer trace.EndCall(ctx)

	return trace.SetCallStatus(ctx, s.append(value, false))
}

func (s *store) append(value IndexMarshaller, processed bool) error {
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

func (s *store) appendEntry(e entry) {
	if s.index == nil {
		s.index = make(map[string]int)
	}
	s.index[e.Key] = len(s.entries)
	s.entries = append(s.entries, e)
}

func (s *store) Get(ctx context.Context, key string, value IndexMarshaller) error {
	ctx = trace.StartCall(ctx, "store.Get")
	defer trace.EndCall(ctx)

	if i, ok := s.index[key]; ok {
		return trace.SetCallStatus(ctx, value.UnmarshalRecord(s.entries[i].Data))
	}

	return nil
}

func (s *store) GetAll(ctx context.Context) *Cursor {
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

func (s *store) GetUnprocessed(ctx context.Context) *Cursor {
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

func (s *store) MarkProcessed(ctx context.Context, recs []IndexMarshaller) error {
	ctx = trace.StartCall(ctx, "store.MarkProcessed")
	defer trace.EndCall(ctx)

	for _, rec := range recs {
		if err := s.append(rec, true); err != nil {
			return trace.SetCallStatus(ctx, err)
		}
	}

	return nil
}

func (s *store) restore(r io.Reader) error {
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
