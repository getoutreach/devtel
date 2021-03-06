// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the logic for matching devspace hooks across CLI runs.

package devspace

import (
	"context"
	"fmt"

	"github.com/getoutreach/devtel/internal/store"
	"github.com/getoutreach/gobox/pkg/trace"
)

// eventBag is internal tracked data bag data from store.
type eventBag map[string]interface{}

func (b eventBag) MarshalRecord(addField func(name string, value interface{})) {
	for k, v := range b {
		if k == "__key" {
			continue
		}
		addField(k, v)
	}
}

// Key returns the key for the event (in eventBag).
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

// EventTracker is responsible for matching events, calculating durations and storing the data in a store.
// It also handles the processing of the events on Flush.
type EventTracker struct {
	s store.Store
	p Processor
}

// Tracker is the entry interface into event tracking, matching and processing.
type Tracker interface {
	Track(context.Context, *Event)
	Flush(context.Context) error
}

// NewTracker creates a new DevspaceTracker.
func NewTracker(p Processor, s store.Store) *EventTracker {
	return &EventTracker{
		s: s,
		p: p,
	}
}

// Track stores and matches an event.
func (t *EventTracker) Track(ctx context.Context, event *Event) {
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

// Flush processes the events in the store.
func (t *EventTracker) Flush(ctx context.Context) error {
	ctx = trace.StartCall(ctx, "tracker.Flush")
	defer trace.EndCall(ctx)

	cursor := t.s.GetUnprocessed(ctx)
	// Generics are bad. bad. We don't want generics.
	var events []store.IndexMarshaller
	var toProcess []interface{}

	for cursor.Next() {
		b := make(eventBag)
		if err := cursor.Value(&b); err != nil {
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

// tryGetBeforeHook tries to get the before hook event for given event.
func (t *EventTracker) tryGetBeforeHook(ctx context.Context, event *Event) *Event {
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

// combineEvents calculates the duration and sets it on the after event.
func (*EventTracker) combineEvents(before, after *Event) *Event {
	after.Duration = after.Timestamp - before.Timestamp

	return after
}
