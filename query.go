package pomdb

import (
	"reflect"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type Query struct {
	Model     interface{}
	Field     string
	Value     any
	Filter    *QueryFilter
	Limit     int
	NextToken string
}

type QueryFilter int

const (
	QueryEqual QueryFilter = iota
	QueryGreaterThan
	QueryLessThan
	QueryBetween
)

const (
	QueryLimitDefault  int         = 100
	QueryFilterDefault QueryFilter = QueryEqual
)

// FilterResults filters the results of a query based on the query filter.
func (q *Query) Compare(obj types.Object, idx *IndexField) (bool, error) {
	ifc, err := decodeIndexPrefix(*obj.Key, *idx)
	if err != nil {
		return false, err
	}

	iValOf := reflect.ValueOf(ifc)
	qValOf := reflect.ValueOf(q.Value)
	comp := *q.Filter

	if idx.FieldType != iValOf.Type() || idx.FieldType != qValOf.Type() {
		return false, nil
	}

	switch idx.FieldType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i := iValOf.Int()
		q := qValOf.Int()
		switch comp {
		case QueryEqual:
			return i == q, nil
		case QueryBetween:
			return i >= q, nil
		case QueryGreaterThan:
			return i > q, nil
		case QueryLessThan:
			return i < q, nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u := iValOf.Uint()
		q := qValOf.Uint()
		switch comp {
		case QueryEqual:
			return u == q, nil
		case QueryBetween:
			return u >= q, nil
		case QueryGreaterThan:
			return u > q, nil
		case QueryLessThan:
			return u < q, nil
		}
	case reflect.Float32, reflect.Float64:
		f := iValOf.Float()
		q := qValOf.Float()
		switch comp {
		case QueryEqual:
			return f == q, nil
		case QueryBetween:
			return f >= q, nil
		case QueryGreaterThan:
			return f > q, nil
		case QueryLessThan:
			return f < q, nil
		}
	case reflect.String:
		s := iValOf.String()
		q := qValOf.String()
		switch comp {
		case QueryEqual:
			return s == q, nil
		case QueryBetween:
			return s >= q, nil
		case QueryGreaterThan:
			return s > q, nil
		case QueryLessThan:
			return s < q, nil
		}
	case reflect.Struct:
		if iValOf.Type().ConvertibleTo(reflect.TypeOf(Timestamp{})) {
			its := iValOf.Interface().(Timestamp)
			qts := qValOf.Interface().(Timestamp)
			switch comp {
			case QueryBetween:
				return (its.After(qts) && its.Before(qts)) || its.Equal(qts), nil
			case QueryGreaterThan:
				return its.After(qts), nil
			case QueryLessThan:
				return its.Before(qts), nil
			}
		}
	}

	return false, nil
}
