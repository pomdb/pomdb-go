package pomdb

import (
	"log"
	"reflect"
	"strings"

	"github.com/gertd/go-pluralize"
	"github.com/iancoleman/strcase"
)

var pluralizer = pluralize.NewClient()

// getCollectionName returns the name of the collection for the given model,
// which is the plural form of the model's name in snake case.
func getCollectionName(i interface{}) string {
	// Get the type of i, dereferencing if it's a pointer
	rt := reflect.TypeOf(i)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	// Convert the name to snake case and pluralize it
	name := pluralizer.Plural(strcase.ToSnake(rt.Name()))

	// Log the original and final names
	log.Printf("GetCollectionName: %s -> %s", rt.Name(), name)

	// Return the pluralized, snake_case name
	return name
}

type IndexFieldValue struct {
	Field string
	Value string
}

// getIndexFieldValues returns the index fields and values for the given model.
func getIndexFieldValues(rv reflect.Value) []IndexFieldValue {
	var indexFields []IndexFieldValue

	for j := 0; j < rv.Elem().NumField(); j++ {
		field := rv.Elem().Type().Field(j)
		value := rv.Elem().Field(j).String()
		if strings.Contains(field.Tag.Get("pomdb"), "index") {
			tagname := field.Tag.Get("json")

			log.Printf("model has unique field: %s", tagname)

			indexFields = append(indexFields, IndexFieldValue{
				Field: tagname,
				Value: value,
			})
		}
	}

	return indexFields
}

type ErrInvalidModelField struct {
	Message string
}

func (e *ErrInvalidModelField) Error() string {
	return e.Message
}

// validateModelFields validates the fields of the given model.
func validateModelFields(i interface{}) *ErrInvalidModelField {
	// Get the type of i, dereferencing if it's a pointer
	rt := reflect.TypeOf(i)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	for j := 0; j < rt.NumField(); j++ {
		field := rt.Field(j)

		switch field.Name {
		case "ID":
			if field.Type.String() != "pomdb.ObjectID" {
				return &ErrInvalidModelField{
					Message: "Record ID field must be a PomDB ObjectID",
				}
			}
		case "CreatedAt":
			if field.Type.String() != "pomdb.Timestamp" {
				return &ErrInvalidModelField{
					Message: "CreatedAt field must be a PomDB Timestamp",
				}
			}
		case "UpdatedAt":
			if field.Type.String() != "pomdb.Timestamp" {
				return &ErrInvalidModelField{
					Message: "UpdatedAt field must be a PomDB Timestamp",
				}
			}
		case "DeletedAt":
			if field.Type.String() != "pomdb.Timestamp" {
				return &ErrInvalidModelField{
					Message: "DeletedAt field must be a PomDB Timestamp",
				}
			}
		}
	}

	return nil
}
