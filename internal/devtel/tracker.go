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
type tracker struct {
	s store.Store
	p Processor
}

type Tracker interface {
	Restore(io.Reader) error

	Record(Event)
	Flush() error
}

func NewWithWriter(w io.Writer, p Processor) Tracker {
	return &tracker{
		s: store.NewWithWriter(func(e interface{}) string { return e.(Event).Hook }, w),
		p: p,
	}
}

func (t *tracker) Record(event Event) {
	if before := t.tryGetBeforeHook(event.Hook); before != nil {
		event = t.combineEvents(*before, event)
	}

	if err := t.s.Append(event); err != nil {
		panic(err)
	}
}

func (t *tracker) Flush() error {
	events := t.s.GetUnprocessed()

	err := t.p.ProcessRecords(events)
	if err != nil {
		return err
	}

	return t.s.MarkProcessed(events)
}

func (t *tracker) Restore(reader io.Reader) error {
	return t.s.Restore(reader, func(m map[string]interface{}) interface{} {
		return Event{
			Hook:      m["hook"].(string),
			Timestamp: int64(m["timestamp"].(float64)),
		}
	})
}

func (t *tracker) tryGetBeforeHook(name string) *Event {
	beforeHook := getBeforeHook(name)
	if beforeHook == "" {
		return nil
	}

	val := t.s.Get(beforeHook).(Event)
	return &val
}

func (*tracker) combineEvents(before, after Event) Event {
	after.Duration = after.Timestamp - before.Timestamp

	return after
}
