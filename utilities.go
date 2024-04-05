package pomdb

import (
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

func dereferenceStruct(i interface{}) (reflect.Value, error) {
	rv := reflect.ValueOf(i)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return reflect.Value{}, errors.New("[Error] DereferenceStruct: model must be a non-nil pointer")
	}

	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return reflect.Value{}, errors.New("[Error] DereferenceStruct: model must be a pointer to a struct")
	}

	if hasPomdbModel(elem) {
		// pomdb.Model is present and assumed to be correctly structured
		return elem, nil
	}

	// Check root level fields and return the dereferenced struct
	if err := checkRootLevelFields(elem, managedTags); err != nil {
		return reflect.Value{}, err
	}

	return elem, nil
}

func hasPomdbModel(v reflect.Value) bool {
	typ := v.Type()
	for j := 0; j < v.NumField(); j++ {
		fieldType := typ.Field(j)
		pomdbModelName := "Model"
		pomdbPkgPath := "github.com/pomdb/pomdb-go"
		if fieldType.Anonymous && fieldType.Type.Name() == pomdbModelName && fieldType.Type.PkgPath() == pomdbPkgPath {
			return true
		}
	}
	return false
}

func checkRootLevelFields(v reflect.Value, rootTags map[string]bool) error {
	typ := v.Type()
	idFieldFound := false

	for j := 0; j < v.NumField(); j++ {
		field := v.Field(j)
		fieldType := typ.Field(j)
		tagValue := fieldType.Tag.Get("pomdb")
		tagParts := strings.Split(tagValue, ",")

		for _, tagPart := range tagParts {
			if rootTags[tagPart] {
				if tagPart == "id" && fieldType.Type.Name() == "ObjectID" {
					idFieldFound = true
				}
				if err := checkSettable(field, fieldType.Name); err != nil {
					return err
				}
			}
		}
	}

	if !idFieldFound {
		return errors.New("[Error] CheckRootLevelFields: model must have an 'id' field of type 'ObjectID'")
	}

	return nil
}

func checkSettable(field reflect.Value, fieldName string) error {
	if !field.CanSet() {
		if isExported := unicode.IsUpper([]rune(fieldName)[0]); !isExported {
			return fmt.Errorf("[Error] CheckSettable: field '%s' is not exported and therefore not settable", fieldName)
		}
		if field.Kind() == reflect.Ptr && field.IsNil() {
			return fmt.Errorf("[Error] CheckSettable: field '%s' is a nil pointer and not settable", fieldName)
		}
	}
	return nil
}

func getFieldByName(v reflect.Value, fieldName string) *reflect.Value {
	field := v.FieldByName(fieldName)
	if field.IsValid() {
		return &field
	}
	return nil
}

// tagContains checks if the tag string contains all the keys in the provided slice.
// It supports both simple tags and key-value pairs.
func tagContains(tagValue string, keys []string) bool {
	tags := strings.Split(tagValue, ",")

	// Create a map for quick lookup of keys
	tagMap := make(map[string]bool)
	for _, tag := range tags {
		key := strings.SplitN(strings.TrimSpace(tag), "=", 2)[0]
		tagMap[key] = true
	}

	// Check if all of the specified keys are present
	for _, key := range keys {
		if _, exists := tagMap[key]; !exists {
			return false
		}
	}

	return true
}

// encodeIndexPrefix returns the index path for the given field name and value.
func encodeIndexPrefix(collection, field, value string, unique bool) (string, error) {
	// Encode the index field value in base64
	code := base64.StdEncoding.EncodeToString([]byte(value))

	if len(code) > 1024 {
		return "", fmt.Errorf("[Error] encodeIndexKey: index %s with value %s is > 1024 bytes", field, value)
	}

	if unique {
		// Create the key path for the unique index item
		return collection + "/indexes/unique/" + field + "/" + code, nil
	} else {
		// Create the key path for the shared index item
		return collection + "/indexes/shared/" + field + "/" + code, nil
	}
}
