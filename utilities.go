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

// getIdFieldValue returns the value of the ID field for the given model.
func getIdFieldValue(rv reflect.Value) string {
	// Get the value of the ID field
	id := rv.Elem().FieldByName("ID").Interface().(ObjectID)

	// Log the ID value
	log.Printf("GetIdFieldValue: %s", id)

	// Return the ID value
	return id.String()
}

type IndexFieldValue struct {
	ID    string
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
				ID:    getIdFieldValue(rv),
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

// initializeModelFields validates the fields of the given model.
func initializeModelFields(i interface{}) *ErrInvalidModelField {
	rt := reflect.TypeOf(i)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	rv := reflect.ValueOf(i)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	for j := 0; j < rt.NumField(); j++ {
		field := rt.Field(j)

		// Check if the field is an embedded struct
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			// Recursively handle fields of the embedded struct
			err := initializeModelFields(rv.Field(j).Addr().Interface())
			if err != nil {
				return err
			}
			continue
		}

		switch field.Name {
		case "ID":
			if field.Type.String() != "pomdb.ObjectID" && field.Type.String() != "ObjectID" {
				return &ErrInvalidModelField{Message: "Record ID field must be a PomDB ObjectID"}
			}
			rv.Field(j).Set(reflect.ValueOf(NewObjectID()))
		case "CreatedAt", "UpdatedAt", "DeletedAt":
			if field.Type.String() != "pomdb.Timestamp" && field.Type.String() != "Timestamp" {
				return &ErrInvalidModelField{Message: field.Name + " field must be a PomDB Timestamp"}
			}
			if field.Name == "DeletedAt" {
				rv.Field(j).Set(reflect.ValueOf(NilTimestamp()))
			} else {
				rv.Field(j).Set(reflect.ValueOf(NewTimestamp()))
			}
		}
	}

	return nil
}
