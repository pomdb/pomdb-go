package pomdb

type Query struct {
	Model interface{}
	Field string
	Value string
	Flags []string
}

type QueryFlag string

const QueryFlagContains QueryFlag = "contains"
