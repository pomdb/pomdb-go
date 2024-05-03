package pomdb

type Query struct {
	Model     interface{}
	Field     string
	Value     string
	Filter    QueryFilter
	Limit     int32
	NextToken string
}

type QueryFilter int

const (
	QueryBetween QueryFilter = iota
	QueryGreaterThan
	QueryLessThan
)

const (
	QueryLimitDefault int32 = 100
)
