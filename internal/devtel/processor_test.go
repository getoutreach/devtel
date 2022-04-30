package devtel

type testProcessor struct {
	lastBatch []interface{}
}

func (p *testProcessor) ProcessRecords(events []interface{}) error {
	p.lastBatch = events

	return nil
}
