package devspace

import "context"

type Processor interface {
	ProcessRecords(context.Context, []interface{}) error
}
