// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains Processor implementation. It wraps the client.SendEvents for use in tracker.

package telefork

import "context"

// Processor wraps the Telefork Client for use in a Tracker.
type Processor struct {
	client *Client
}

// NewProcessor returns a new Telefork Processor.
func NewProcessor(appName, apiKey string) *Processor {
	return &Processor{
		client: NewClient(appName, apiKey),
	}
}

// ProcessRecords sends the given events to Telefork.
func (p *Processor) ProcessRecords(ctx context.Context, events []interface{}) error {
	return p.client.SendEvents(ctx, events)
}
