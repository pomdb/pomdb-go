package pomdb

type Query struct {
	Model interface{}
	Field string
	Value string
	Limit int32
	Token string
}

type QueryFilter string

const (
	QueryLimitDefault int32 = 100
)
