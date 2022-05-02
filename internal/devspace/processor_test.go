package devspace

import "context"

type testProcessor struct {
	lastBatch []interface{}
}

func (p *testProcessor) ProcessRecords(ctx context.Context, events []interface{}) error {
	p.lastBatch = events

	return nil
}
