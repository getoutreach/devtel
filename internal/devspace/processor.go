// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains event processor definition.
// The processors are used to send events to Telefork (at least the TeleforkProcessor does that).

package devspace

import "context"

// Processor is the interface for processing stored events.
type Processor interface {
	ProcessRecords(context.Context, []interface{}) error
}
