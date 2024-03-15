package pomdb

import (
	"fmt"
	"strings"
)

type Query struct {
	Model      interface{}
	FieldName  string
	FieldValue string
	Matches    []map[string]string
	Filter     QueryFilter
}

type QueryFilter string

const QueryFilterContains QueryFilter = "contains"
const QueryFilterEquals QueryFilter = "equals"
const QueryFilterStartsWith QueryFilter = "startsWith"
const QueryFilterEndsWith QueryFilter = "endsWith"
const QueryFilterGreaterThan QueryFilter = "greaterThan"
const QueryFilterLessThan QueryFilter = "lessThan"

func (q *Query) FilterMatches(k []map[string]string) error {
	for _, v := range k {
		switch q.Filter {
		case QueryFilterContains:
			if strings.Contains(v["value"], q.FieldValue) {
				q.Matches = append(q.Matches, v)
			}
		case QueryFilterEquals:
			if v["value"] == q.FieldValue {
				q.Matches = append(q.Matches, v)
			}
		case QueryFilterStartsWith:
			if strings.HasPrefix(v["value"], q.FieldValue) {
				q.Matches = append(q.Matches, v)
			}
		case QueryFilterEndsWith:
			if strings.HasSuffix(v["value"], q.FieldValue) {
				q.Matches = append(q.Matches, v)
			}
		case QueryFilterGreaterThan:
			if v["value"] > q.FieldValue {
				q.Matches = append(q.Matches, v)
			}
		case QueryFilterLessThan:
			if v["value"] < q.FieldValue {
				q.Matches = append(q.Matches, v)
			}
		default:
			return fmt.Errorf("[Error] FilterMatches: invalid filter '%s'", q.Filter)
		}
	}

	return nil
}
