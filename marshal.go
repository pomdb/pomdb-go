package pomdb

import (
	"encoding/json"
	"errors"
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

// GetModelName returns the name of the model for the given collection,
// which is the singular form of the collection name in camel case.
func GetModelName(collection string) string {
	return strcase.ToCamel(pluralizer.Singular(collection))
}

// MarshalJSON takes a model and returns a JSON representation of it,
// converting the field names to their database equivalents.
func MarshalJSON(i interface{}) ([]byte, error) {
	// Reflect the type and value of i, dereferencing if it's a pointer
	t := reflect.TypeOf(i)
	v := reflect.ValueOf(i)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	// Return an error if i is not a struct
	if t.Kind() != reflect.Struct {
		return nil, errors.New("input must be a struct or pointer to a struct")
	}

	// Initialize the data map
	data := make(map[string]interface{})

	for i := 0; i < t.NumField(); i++ {
		typeField := t.Field(i)
		field := v.Field(i)

		// Use the json tag as the key if it's set, otherwise fall back to snake_case
		jsonKey := typeField.Tag.Get("json")
		if jsonKey == "" {
			jsonKey = strcase.ToSnake(typeField.Name)
		}

		data[jsonKey] = field.Interface()
	}

	return json.Marshal(data)
}

// UnmarshalJSON takes a JSON representation of a model and returns
// a model with the field names converted to their Go equivalents.
func UnmarshalJSON(data []byte, i interface{}) error {
	// Check that i is a pointer
	if reflect.TypeOf(i).Kind() != reflect.Ptr {
		return errors.New("input must be a pointer")
	}

	// Unmarshal JSON into a temporary map
	tempData := make(map[string]interface{})
	if err := json.Unmarshal(data, &tempData); err != nil {
		return err
	}

	// Create a map for JSON keys to struct field indexes
	t := reflect.TypeOf(i).Elem()
	fieldIndexes := make(map[string]int)
	for i := 0; i < t.NumField(); i++ {
		typeField := t.Field(i)

		// Determine the JSON key for the field
		jsonKey := typeField.Tag.Get("json")
		if jsonKey == "" {
			jsonKey = strcase.ToSnake(typeField.Name)
		}

		fieldIndexes[jsonKey] = i
	}

	// Set the corresponding struct fields
	v := reflect.ValueOf(i).Elem()
	for key, value := range tempData {
		if index, ok := fieldIndexes[key]; ok {
			field := v.Field(index)
			if field.CanSet() {
				field.Set(reflect.ValueOf(value))
			}
		}
	}

	return nil
}
