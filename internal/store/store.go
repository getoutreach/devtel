package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/pkg/errors"
)

type entry struct {
	Key       string                 `json:"key"`
	Data      map[string]interface{} `json:"data"`
	Processed bool                   `json:"processed,omitempty"`
}

type Store interface {
	Init() error

	Append(value IndexMarshaller) error

	Get(key string, value IndexMarshaller) error
	GetAll() *Cursor
	GetUnprocessed() *Cursor

	MarkProcessed([]IndexMarshaller) error
}

type store struct {
	logDir     string
	logPath    string
	logFS      fs.FS
	openAppend func(path string) (io.WriteCloser, error)

	entries []entry
	index   map[string]int
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
		return err
	}

	if s.logPath == "" {
		s.logPath = filepath.Join(s.logDir, fmt.Sprintf("%d.log", time.Now().Unix()))
	}

	return nil
}

func (s *store) Append(value IndexMarshaller) error {
	return s.append(value, false)
}

func (s *store) append(value IndexMarshaller, processed bool) error {
	val := make(map[string]interface{})

	value.MarshalRecord(addField(val))

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

func (s *store) Get(key string, value IndexMarshaller) error {
	if i, ok := s.index[key]; ok {
		return value.UnmarshalRecord(s.entries[i].Data)
	}

	return nil
}

func (s *store) GetAll() *Cursor {
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

func (s *store) GetUnprocessed() *Cursor {
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

func (s *store) MarkProcessed(recs []IndexMarshaller) error {
	for _, rec := range recs {
		if err := s.append(rec, true); err != nil {
			return err
		}
	}

	return nil
}

func (s *store) restore(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var e entry

		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			fmt.Println("Scanner.Text:")
			fmt.Println(scanner.Text())
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
