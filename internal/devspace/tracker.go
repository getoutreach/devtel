package devspace

import (
	"context"
	"fmt"

	"github.com/getoutreach/devtel/internal/store"
	"github.com/getoutreach/gobox/pkg/trace"
)

type tracker struct {
	s store.Store
	p Processor
}

type Tracker interface {
	Init(context.Context) error

	Track(context.Context, *Event)
	Flush(context.Context) error
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
		o.Store = store.New(&store.Options{})
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

func (t *tracker) Init(ctx context.Context) error {
	ctx = trace.StartCall(ctx, "tracker.Init")
	defer trace.EndCall(ctx)

	return t.s.Init()
}

func (t *tracker) Track(ctx context.Context, event *Event) {
	ctx = trace.StartCall(ctx, "tracker.Track")
	defer trace.EndCall(ctx)

	if before := t.tryGetBeforeHook(event); before != nil {
		event = t.combineEvents(before, event)
	}

	if err := t.s.Append(event); err != nil {
		panic(err)
	}
}

func (t *tracker) Flush(ctx context.Context) error {
	ctx = trace.StartCall(ctx, "tracker.Flush")
	defer trace.EndCall(ctx)

	cursor := t.s.GetUnprocessed()
	// Generics are bad. bad. We don't want generics.
	var events []store.IndexMarshaller
	var toProcess []interface{}

	for cursor.Next() {
		var event Event
		if err := cursor.Value(&event); err != nil {
			// TODO: probably should logs something here
			continue
		}
		events = append(events, &event)
		toProcess = append(toProcess, &event)
	}

	err := t.p.ProcessRecords(ctx, toProcess)
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

	var val Event
	if err := t.s.Get(beforeKey, &val); err != nil {
		return nil
	}
	return &val
}

func (*tracker) combineEvents(before, after *Event) *Event {
	after.Duration = after.Timestamp - before.Timestamp

	return after
}
