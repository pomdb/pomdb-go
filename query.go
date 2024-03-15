package pomdb

type Query struct {
	Model      interface{}
	FieldName  string
	FieldValue string
	Filter     QueryFilter
}

type QueryFilter string

const QueryFilterContains QueryFilter = "contains"
const QueryFilterEquals QueryFilter = "equals"
const QueryFilterStartsWith QueryFilter = "startsWith"
const QueryFilterEndsWith QueryFilter = "endsWith"
const QueryFilterGreaterThan QueryFilter = "greaterThan"
const QueryFilterLessThan QueryFilter = "lessThan"
