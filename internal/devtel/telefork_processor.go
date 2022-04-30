package devtel

import "github.com/getoutreach/devtel/internal/telefork"

type teleforkProcessor struct {
	client *telefork.Client
}

func (p *teleforkProcessor) ProcessRecords(events []interface{}) error {
	return p.client.SendEvents(events)
}
