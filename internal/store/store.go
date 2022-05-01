package store

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type entry struct {
	Key       string      `json:"key"`
	Data      interface{} `json:"data"`
	Processed bool        `json:"processed,omitempty"`
}

type Store interface {
	Init() error

	Append(value interface{}) error

	Get(key string) interface{}
	GetAll() []interface{}
	GetUnprocessed() []interface{}

	MarkProcessed([]interface{}) error
}

type store struct {
	logDir     string
	logPath    string
	logFS      fs.FS
	openAppend func(path string) (io.WriteCloser, error)

	entries []entry
	index   map[string]int

	extractKey       func(interface{}) string
	restoreConverter func(map[string]interface{}) interface{}
}

type Options struct {
	LogDir     string
	LogFS      fs.FS
	OpenAppend func(path string) (io.WriteCloser, error)

	RestoreConverter func(map[string]interface{}) interface{}
}

func New(extractKey func(interface{}) string, opts *Options) Store {
	if opts.LogDir == "" {
		opts.LogDir = filepath.Join(os.TempDir(), "devtel")
	}

	//nolint
	os.MkdirAll(opts.LogDir, 0o755)

	if opts.LogFS == nil {
		opts.LogFS = os.DirFS(opts.LogDir)
	}

	if opts.OpenAppend == nil {
		opts.OpenAppend = func(path string) (io.WriteCloser, error) {
			return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		}
	}

	return &store{
		logDir:     opts.LogDir,
		logFS:      opts.LogFS,
		openAppend: opts.OpenAppend,

		extractKey:       extractKey,
		restoreConverter: opts.RestoreConverter,
	}
}

func (s *store) Init() error {
	err := fs.WalkDir(s.logFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		log.Printf("Found log file: %s\n", filepath.Join(s.logDir, path))

		f, err := s.logFS.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		log.Printf("Restoring log file: %s\n", filepath.Join(s.logDir, path))
		if err := s.restore(f); err != nil {
			return err
		}

		log.Printf("Setting log file: %s\n", filepath.Join(s.logDir, path))
		s.logPath = filepath.Join(s.logDir, path)

		return nil
	})

	if err != nil {
		return err
	}

	if s.logPath == "" {
		s.logPath = filepath.Join(s.logDir, fmt.Sprintf("%d.log", time.Now().Unix()))
		log.Printf("Setting log file: %s\n", filepath.Join(s.logDir, s.logPath))
	}

	return nil
}

func (s *store) Append(value interface{}) error {
	log.Printf("Append: %s\n", s.extractKey(value))

	return s.append(value, false)
}

func (s *store) append(value interface{}, processed bool) error {
	e := entry{s.extractKey(value), value, processed}
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

func (s *store) Get(key string) interface{} {
	log.Printf("Get: %s\n", key)

	if i, ok := s.index[key]; ok {
		return s.entries[i].Data
	}
	return nil
}

func (s *store) GetAll() []interface{} {
	log.Printf("GetAll\n")

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
	log.Printf("GetUnprocessed\n")

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
	log.Printf("MarkProcessed\n")

	for _, rec := range recs {
		if err := s.append(rec, true); err != nil {
			return err
		}
	}

	return nil
}

func (s *store) restore(r io.Reader) error {
	log.Printf("Restore\n")

	dec := json.NewDecoder(r)

	for dec.More() {
		var e entry
		if err := dec.Decode(&e); err != nil {
			return err
		}

		if e.Key == "" {
			continue
		}
		if e.Data == nil {
			continue
		}

		// JSON transforms into map[string]interface{}
		if s.restoreConverter != nil {
			e.Data = s.restoreConverter(e.Data.(map[string]interface{}))
		} else {
			e.Data = e.Data.(map[string]interface{})
		}

		s.appendEntry(e)
	}

	return nil
}
