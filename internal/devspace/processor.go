package devspace

type Processor interface {
	ProcessRecords([]interface{}) error
}
