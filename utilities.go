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
		key := strings.Split(strings.TrimSpace(tag), "=")[0]
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
func encodeIndexPrefix(collection, field, value string, idxtype IndexType) (string, error) {
	// Encode the index field value in base64
	code := base64.StdEncoding.EncodeToString([]byte(value))

	if len(code) > 1024 {
		return "", fmt.Errorf("[Error] encodeIndexPrefix: index %s with value %s is > 1024 bytes", field, value)
	}

	switch idxtype {
	case RangedIndex:
		return collection + "/indexes/ranged/" + field + "/" + code, nil
	case UniqueIndex:
		return collection + "/indexes/unique/" + field + "/" + code, nil
	case SharedIndex:
		return collection + "/indexes/shared/" + field + "/" + code, nil
	default:
		return "", errors.New("[Error] encodeIndexPrefix: invalid index type")
	}
}

func stringifyFieldValue(field reflect.Value, ftype reflect.StructField) (string, error) {
	// Check if the field is embedded
	if ftype.Anonymous {
		return "", nil // Skip processing for embedded structs
	}

	// Example of handling Timestamp or other specific types
	if ftype.Type.ConvertibleTo(reflect.TypeOf(Timestamp{})) {
		ts := field.Interface().(Timestamp)
		return ts.String(), nil
	}

	// Handle basic types
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%v", field.Interface()), nil
	case reflect.String:
		return field.String(), nil
	}

	return "", fmt.Errorf("unsupported field type %s", ftype.Type)
}
