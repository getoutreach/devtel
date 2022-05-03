// Copyright 2022 Outreach Corporation. All Rights Reserved.

package devspace

import (
	"context"
	"encoding/json"
)

type testProcessor struct {
	lastBatch []string
}

func (p *testProcessor) ProcessRecords(ctx context.Context, events []interface{}) error {
	p.lastBatch = make([]string, 0, len(events))
	for _, e := range events {
		b, err := json.Marshal(e)
		if err != nil {
			return err
		}
		p.lastBatch = append(p.lastBatch, string(b))
	}

	return nil
}
