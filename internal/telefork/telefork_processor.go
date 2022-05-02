package telefork

import "context"

type Processor struct {
	client *Client
}

func NewProcessor(appName, apiKey string) *Processor {
	return &Processor{
		client: NewClient(appName, apiKey),
	}
}

func (p *Processor) ProcessRecords(ctx context.Context, events []interface{}) error {
	return p.client.SendEvents(events)
}
