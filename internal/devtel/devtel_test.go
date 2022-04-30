package devtel

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/getoutreach/devtel/internal/store"
	"github.com/stretchr/testify/assert"
)

type Event struct {
	Hook string `json:"hook"`

	Timestamp int64 `json:"timestamp"`
	Duration  int64 `json:"duration_ms,omitempty"`
}
type recorder struct {
	s store.Store
}

type Recorder interface {
	Record(Event)
	Restore(io.Reader) error
}

func NewWithWriter(w io.Writer) Recorder {
	return &recorder{s: store.NewWithWriter(func(e interface{}) string { return e.(Event).Hook }, w)}
}

func (r *recorder) Record(data Event) {
	if before := r.tryGetBeforeHook(data.Hook); before != nil {
		data = r.combineEvents(*before, data)
	}

	r.s.Append(data)
}

func (r *recorder) Restore(reader io.Reader) error {
	return r.s.Restore(reader, func(m map[string]interface{}) interface{} {
		return Event{
			Hook:      m["hook"].(string),
			Timestamp: int64(m["timestamp"].(float64)),
		}
	})
}

func (r *recorder) tryGetBeforeHook(name string) *Event {
	beforeHook := getBeforeHook(name)
	if beforeHook == "" {
		return nil
	}

	val := r.s.Get(beforeHook).(Event)
	return &val
}

func (*recorder) combineEvents(before, after Event) Event {
	after.Duration = after.Timestamp - before.Timestamp

	return after
}

func TestEventWrittenToBuffer(t *testing.T) {
	var buff bytes.Buffer
	r := NewWithWriter(&buff)

	r.Record(Event{Hook: "before:deploy", Timestamp: 2147483605})

	expected := `{"key":"before:deploy","data":{"hook":"before:deploy","timestamp":2147483605}}` + "\n"

	assert.Equal(t, expected, buff.String())
}

func TestEventMatched(t *testing.T) {
	var buff bytes.Buffer
	r := NewWithWriter(&buff)

	r.Record(Event{Hook: "before:deploy", Timestamp: 2147483605})
	r.Record(Event{Hook: "after:deploy", Timestamp: 2147483647})

	expected := "" + // This makes it nicely arranged
		`{"key":"before:deploy","data":{"hook":"before:deploy","timestamp":2147483605}}` + "\n" +
		`{"key":"after:deploy","data":{"hook":"after:deploy","timestamp":2147483647,"duration_ms":42}}` + "\n"

	assert.Equal(t, expected, buff.String())
}

func TestCanRestoreFromReader(t *testing.T) {
	f := strings.NewReader("" + // This makes it nicely arranged
		`{"key":"before:deploy","data":{"hook":"before:deploy","timestamp":2147483646}}` + "\n")

	var buff bytes.Buffer
	r := NewWithWriter(&buff)

	if err := r.Restore(f); err != nil {
		t.Error(err)
	}
	r.Record(Event{Hook: "after:deploy", Timestamp: 2147483647})

	expected := `{"key":"after:deploy","data":{"hook":"after:deploy","timestamp":2147483647,"duration_ms":1}}` + "\n"
	assert.Equal(t, expected, buff.String())
}
