package devspace

import (
	"fmt"
	"log"

	"github.com/getoutreach/devtel/internal/store"
)

type tracker struct {
	s store.Store
	p Processor
}

type Tracker interface {
	Init() error

	Track(*Event)
	Flush() error
}

type Options struct {
	Store store.Store
}

func NewTracker(p Processor, opts ...func(*Options)) Tracker {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	if o.Store == nil {
		o.Store = store.New(eventKey, &store.Options{
			RestoreConverter: eventFromMap,
		})
	}

	return &tracker{
		s: o.Store,
		p: p,
	}
}

func WithStore(s store.Store) func(opts *Options) {
	return func(opts *Options) {
		opts.Store = s
	}
}

func (t *tracker) Init() error {
	return t.s.Init()
}

func (t *tracker) Track(event *Event) {
	if before := t.tryGetBeforeHook(event); before != nil {
		log.Printf("Found before event: %s\n", before.Hook)
		event = t.combineEvents(before, event)
	}

	if err := t.s.Append(event); err != nil {
		panic(err)
	}
}

func (t *tracker) Flush() error {
	log.Println("Flush")
	events := t.s.GetUnprocessed()

	err := t.p.ProcessRecords(events)
	if err != nil {
		return err
	}

	return t.s.MarkProcessed(events)
}

func (t *tracker) tryGetBeforeHook(event *Event) *Event {
	beforeHook := getBeforeHook(event.Hook)
	if beforeHook == "" {
		return nil
	}

	beforeKey := beforeHook
	if event.ExecutionID != "" {
		beforeKey = fmt.Sprintf("%s_%s", event.ExecutionID, beforeHook)
	}

	val := t.s.Get(beforeKey)
	if val == nil {
		return nil
	}
	return val.(*Event)
}

func (*tracker) combineEvents(before, after *Event) *Event {
	after.Duration = after.Timestamp - before.Timestamp

	return after
}
