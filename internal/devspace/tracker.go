package devspace

import (
	"context"
	"fmt"

	"github.com/getoutreach/devtel/internal/store"
	"github.com/getoutreach/gobox/pkg/trace"
)

type eventBag map[string]interface{}

func (b eventBag) MarshalRecord(addField func(name string, value interface{})) {
	for k, v := range b {
		if k == "__key" {
			continue
		}
		addField(k, v)
	}
}

func (b eventBag) Key() string {
	var key string

	if v, ok := b["execution_id"]; ok {
		key = v.(string) + "_"
	}

	if v, ok := b["hook"]; ok {
		key += v.(string)
	}

	return key
}

func (b eventBag) UnmarshalRecord(data map[string]interface{}) error {
	for k, v := range data {
		b[k] = v
	}

	return nil
}

type tracker struct {
	s store.Store
	p Processor
}

type Tracker interface {
	Init(context.Context) error

	AddDefaultField(string, interface{})

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

	return trace.SetCallStatus(ctx, t.s.Init(ctx))
}

func (t *tracker) AddDefaultField(k string, v interface{}) {
	t.s.AddDefaultField(k, v)
}

func (t *tracker) Track(ctx context.Context, event *Event) {
	ctx = trace.StartCall(ctx, "tracker.Track")
	defer trace.EndCall(ctx)

	if before := t.tryGetBeforeHook(ctx, event); before != nil {
		event = t.combineEvents(before, event)
	}

	if err := t.s.Append(ctx, event); err != nil {
		//nolint:errcheck // Why: This is how we track it. There's not much else we should do. Definitely not crashing devspace.
		trace.SetCallStatus(ctx, err)
	}
}

func (t *tracker) Flush(ctx context.Context) error {
	ctx = trace.StartCall(ctx, "tracker.Flush")
	defer trace.EndCall(ctx)

	cursor := t.s.GetUnprocessed(ctx)
	// Generics are bad. bad. We don't want generics.
	var events []store.IndexMarshaller
	var toProcess []interface{}

	for cursor.Next() {
		b := make(eventBag)
		if err := cursor.Value(&b); err != nil {
			// TODO: probably should logs something here
			continue
		}

		events = append(events, &b)
		toProcess = append(toProcess, &b)
	}

	err := t.p.ProcessRecords(ctx, toProcess)
	if err != nil {
		return trace.SetCallStatus(ctx, err)
	}

	return trace.SetCallStatus(ctx, t.s.MarkProcessed(ctx, events))
}

func (t *tracker) tryGetBeforeHook(ctx context.Context, event *Event) *Event {
	beforeHook := getBeforeHook(event.Hook)
	if beforeHook == "" {
		return nil
	}

	beforeKey := beforeHook
	if event.ExecutionID != "" {
		beforeKey = fmt.Sprintf("%s_%s", event.ExecutionID, beforeHook)
	}

	var val Event
	if err := t.s.Get(ctx, beforeKey, &val); err != nil {
		return nil
	}
	return &val
}

func (*tracker) combineEvents(before, after *Event) *Event {
	after.Duration = after.Timestamp - before.Timestamp

	return after
}
