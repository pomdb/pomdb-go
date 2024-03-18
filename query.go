package pomdb

import (
	"fmt"
	"strings"
)

type Query struct {
	Model   interface{}
	Field   string
	Value   string
	Matches []map[string]string
	Filter  QueryFilter
	Limit   int32
	Token   string
}

type QueryFilter string

const (
	QueryFilterContains    QueryFilter = "contains"
	QueryFilterEquals      QueryFilter = "equals"
	QueryFilterStartsWith  QueryFilter = "startsWith"
	QueryFilterEndsWith    QueryFilter = "endsWith"
	QueryFilterGreaterThan QueryFilter = "greaterThan"
	QueryFilterLessThan    QueryFilter = "lessThan"
	QueryFilterDefault     QueryFilter = QueryFilterEquals
	QueryLimitDefault      int32       = 100
)

func (q *Query) FilterMatches(k []map[string]string) error {
	for _, v := range k {
		switch q.Filter {
		case QueryFilterContains:
			if strings.Contains(v["value"], q.Value) {
				q.Matches = append(q.Matches, v)
			}
		case QueryFilterEquals:
			if v["value"] == q.Value {
				q.Matches = append(q.Matches, v)
			}
		case QueryFilterStartsWith:
			if strings.HasPrefix(v["value"], q.Value) {
				q.Matches = append(q.Matches, v)
			}
		case QueryFilterEndsWith:
			if strings.HasSuffix(v["value"], q.Value) {
				q.Matches = append(q.Matches, v)
			}
		case QueryFilterGreaterThan:
			if v["value"] > q.Value {
				q.Matches = append(q.Matches, v)
			}
		case QueryFilterLessThan:
			if v["value"] < q.Value {
				q.Matches = append(q.Matches, v)
			}
		default:
			return fmt.Errorf("[Error] FilterMatches: invalid filter '%s'", q.Filter)
		}
	}

	return nil
}
