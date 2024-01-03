package pomdb

import (
	"encoding/json"
	"errors"
	"reflect"

	"github.com/gertd/go-pluralize"
	"github.com/iancoleman/strcase"
)

var pluralizer = pluralize.NewClient()

// GetCollectionName returns the name of the collection for the given model,
// which is the plural form of the model's name in snake case.
func GetCollectionName(i interface{}) string {
	t := reflect.TypeOf(i)
	return pluralizer.Plural(strcase.ToSnake(t.Name()))
}

// GetModelName returns the name of the model for the given collection,
// which is the singular form of the collection name in camel case.
func GetModelName(collection string) string {
	return strcase.ToCamel(pluralizer.Singular(collection))
}

// MarshalJSON takes a model and returns a JSON representation of it,
// converting the field names to their database equivalents.
func MarshalJSON(i interface{}) ([]byte, error) {
	// Ensure the provided interface is a pointer
	if reflect.TypeOf(i).Kind() != reflect.Ptr {
		return nil, errors.New("model must be a pointer to a struct")
	}

	// Dereference the pointer to get the struct value
	structValue := reflect.ValueOf(i).Elem()
	structType := structValue.Type()

	// Initialize the data map
	data := make(map[string]interface{})

	for i := 0; i < structValue.NumField(); i++ {
		field := structValue.Field(i)
		typeField := structType.Field(i)

		// Use the json tag as the key if it's set, otherwise fall back to snake_case
		jsonKey := typeField.Tag.Get("json")
		if jsonKey == "" {
			jsonKey = strcase.ToSnake(typeField.Name) // Convert field name to snake_case
		}

		data[jsonKey] = field.Interface()
	}

	return json.Marshal(data)
}

// UnmarshalJSON takes a JSON representation of a model and returns
// a model with the field names converted to their Go equivalents.
func UnmarshalJSON(data []byte, ptr interface{}) error {
	// Ensure the provided interface is a pointer
	if reflect.TypeOf(ptr).Kind() != reflect.Ptr {
		return errors.New("model must be a pointer to a struct")
	}

	// Temporary map to hold the JSON data
	tempData := make(map[string]interface{})
	err := json.Unmarshal(data, &tempData)
	if err != nil {
		return err
	}

	// Dereference the pointer to get the struct value
	structValue := reflect.ValueOf(ptr).Elem()
	structType := structValue.Type()

	// Create a map to easily look up field indexes by their JSON (snake_case) keys
	fieldMap := make(map[string]int)
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			jsonTag = strcase.ToSnake(field.Name) // Convert field name to snake_case
		}
		fieldMap[jsonTag] = i
	}

	// Iterate over the JSON keys and set the corresponding struct fields
	for key, jsonValue := range tempData {
		if fieldIndex, ok := fieldMap[key]; ok {
			structField := structValue.Field(fieldIndex)
			fieldValue := reflect.ValueOf(jsonValue)
			if structField.Type() == fieldValue.Type() && structField.CanSet() {
				structField.Set(fieldValue)
			}
		}
	}

	return nil
}
