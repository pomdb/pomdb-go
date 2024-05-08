package pomdb

import "github.com/aws/aws-sdk-go-v2/service/s3/types"

type Query struct {
	Model     interface{}
	Field     string
	Value     string
	Filter    *QueryFilter
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

type QueryHandler func(obj types.Object) bool

func (q *Query) GetHandler() QueryHandler {
	switch *q.Filter {
	case QueryBetween:
		return q.FilterBetween
	case QueryGreaterThan:
		return q.FilterGreaterThan
	case QueryLessThan:
		return q.FilterLessThan
	}
	return nil
}

// FilterGreaterThan decodes the value in the object key and returns true of
// the value is greater than the value in the query.
func (q *Query) FilterGreaterThan(obj types.Object) bool {
	val, err := decodeIndexPrefix(*obj.Key)
	if err != nil {
		return false
	}
	return val > q.Value
}

// FilterLessThan decodes the value in the object key and returns true of
// the value is less than the value in the query.
func (q *Query) FilterLessThan(obj types.Object) bool {
	val, err := decodeIndexPrefix(*obj.Key)
	if err != nil {
		return false
	}
	return val < q.Value
}

// FilterBetween decodes the value in the object key and returns true of
// the value is between the values in the query.
func (q *Query) FilterBetween(obj types.Object) bool {
	val, err := decodeIndexPrefix(*obj.Key)
	if err != nil {
		return false
	}
	return val > q.Value && val < q.NextToken
}
