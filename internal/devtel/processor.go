package devtel

type Processor interface {
	ProcessRecords([]interface{}) error
}
