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

// getUniqueFieldMeta returns the unique field and value for the given model.
func getUniqueFieldMeta(rv reflect.Value) (string, string) {
	var uniqueField string
	var uniqueValue string

	for j := 0; j < rv.Elem().NumField(); j++ {
		field := rv.Elem().Type().Field(j)
		value := rv.Elem().Field(j).String()
		if strings.Contains(field.Tag.Get("pomdb"), "unique") {
			tagname := field.Tag.Get("json")

			log.Printf("model has unique field: %s", tagname)

			uniqueField = tagname
			uniqueValue = value
		}
	}

	return uniqueField, uniqueValue
}
