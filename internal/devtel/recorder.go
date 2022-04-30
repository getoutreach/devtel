package devtel

import (
	"io"

	"github.com/getoutreach/devtel/internal/store"
)

type Event struct {
	Hook string `json:"hook"`

	Timestamp int64 `json:"timestamp"`
	Duration  int64 `json:"duration_ms,omitempty"`
}
type recorder struct {
	s store.Store
	p Processor
}

type Recorder interface {
	Restore(io.Reader) error

	Record(Event)
	ProcessRecords() error
}

func NewWithWriter(w io.Writer, p Processor) Recorder {
	return &recorder{
		s: store.NewWithWriter(func(e interface{}) string { return e.(Event).Hook }, w),
		p: p,
	}
}

func (r *recorder) Record(data Event) {
	if before := r.tryGetBeforeHook(data.Hook); before != nil {
		data = r.combineEvents(*before, data)
	}

	if err := r.s.Append(data); err != nil {
		panic(err)
	}
}

func (r *recorder) ProcessRecords() error {
	events := r.s.GetUnprocessed()

	err := r.p.ProcessRecords(events)
	if err != nil {
		return err
	}

	return r.s.MarkProcessed(events)
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
