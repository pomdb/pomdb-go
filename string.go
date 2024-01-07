package pomdb

import (
	"log"
	"reflect"

	"github.com/gertd/go-pluralize"
	"github.com/iancoleman/strcase"
)

var pluralizer = pluralize.NewClient()

// GetCollectionName returns the name of the collection for the given model,
// which is the plural form of the model's name in snake case.
func GetCollectionName(i interface{}) string {
	// Get the type of i, dereferencing if it's a pointer
	typ := reflect.TypeOf(i)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// Convert the name to snake case and pluralize it
	name := pluralizer.Plural(strcase.ToSnake(typ.Name()))

	// Log the original and final names
	log.Printf("GetCollectionName: %s -> %s", typ.Name(), name)

	// Return the pluralized, snake_case name
	return name
}
