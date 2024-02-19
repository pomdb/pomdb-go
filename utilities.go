package pomdb

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"unicode"

	"github.com/gertd/go-pluralize"
	"github.com/iancoleman/strcase"
)

type IndexField struct {
	ID    string
	Field string
	Value string
}

type StructCache struct {
	Collection     string
	IDField        *reflect.Value
	CreatedAtField *reflect.Value
	UpdatedAtField *reflect.Value
	DeletedAtField *reflect.Value
}

func (sc *StructCache) SetNewModelFields() string {
	id := NewObjectID()
	sc.IDField.Set(reflect.ValueOf(id))

	ts := NewTimestamp()
	if sc.CreatedAtField != nil && sc.CreatedAtField.CanSet() {
		sc.CreatedAtField.Set(reflect.ValueOf(ts))
	}
	if sc.UpdatedAtField != nil && sc.UpdatedAtField.CanSet() {
		sc.UpdatedAtField.Set(reflect.ValueOf(ts))
	}
	if sc.DeletedAtField != nil && sc.DeletedAtField.CanSet() {
		sc.DeletedAtField.Set(reflect.ValueOf(NilTimestamp()))
	}

	return id.String()
}

func (sc *StructCache) SetUpdatedAt() {
	if sc.UpdatedAtField != nil && sc.UpdatedAtField.CanSet() {
		sc.UpdatedAtField.Set(reflect.ValueOf(NewTimestamp()))
	}
}

func (sc *StructCache) SetDeletedAt() {
	if sc.DeletedAtField != nil && sc.DeletedAtField.CanSet() {
		sc.DeletedAtField.Set(reflect.ValueOf(NilTimestamp()))
	}
}

// getIndexFieldValues returns the index fields and values for the given model.
func getIndexFieldValues(rv reflect.Value, id string) []IndexField {
	var indexFields []IndexField

	for j := 0; j < rv.NumField(); j++ {
		field := rv.Type().Field(j)
		value := rv.Field(j).String()
		if strings.Contains(field.Tag.Get("pomdb"), "index") {
			tagname := field.Tag.Get("json")

			log.Printf("model has unique field: %s", tagname)

			indexFields = append(indexFields, IndexField{
				ID:    id,
				Field: tagname,
				Value: value,
			})
		}
	}

	return indexFields
}

func dereferenceStruct(i interface{}) (reflect.Value, error) {
	rv := reflect.ValueOf(i)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return reflect.Value{}, errors.New("input must be a non-nil pointer to a struct")
	}

	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return reflect.Value{}, errors.New("input must be a pointer to a struct")
	}

	if hasPomdbModel(elem) {
		// pomdb.Model is present and assumed to be correctly structured
		return elem, nil
	}

	rootTags := map[string]bool{
		"id":         true,
		"created_at": true,
		"updated_at": true,
		"deleted_at": true,
	}

	// Check root level fields and return the dereferenced struct
	if err := checkRootLevelFields(elem, rootTags); err != nil {
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
				if tagPart == "id" {
					idFieldFound = true
				}
				if err := checkSettable(field, fieldType.Name); err != nil {
					return err
				}
			}
		}
	}

	if !idFieldFound {
		return errors.New("required 'id' field not found at the root level")
	}

	return nil
}

func checkSettable(field reflect.Value, fieldName string) error {
	if !field.CanSet() {
		if isExported := unicode.IsUpper([]rune(fieldName)[0]); !isExported {
			return fmt.Errorf("field '%s' is not exported and therefore not settable", fieldName)
		}
		if field.Kind() == reflect.Ptr && field.IsNil() {
			return fmt.Errorf("field '%s' is a nil pointer and not settable", fieldName)
		}
	}
	return nil
}

func buildStructCache(rv reflect.Value) *StructCache {
	sc := &StructCache{}

	// Get the collection name
	sc.Collection = pluralize.NewClient().Plural(strcase.ToSnake(rv.Type().Name()))

	// Log the original and final names
	log.Printf("Collection: %s -> %s", rv.Type().Name(), sc.Collection)

	if hasPomdbModel(rv) {
		// Use fields from embedded pomdb.Model
		sc.IDField = getFieldFromStruct(rv, "ID")
		sc.CreatedAtField = getFieldFromStruct(rv, "CreatedAt")
		sc.UpdatedAtField = getFieldFromStruct(rv, "UpdatedAt")
		sc.DeletedAtField = getFieldFromStruct(rv, "DeletedAt")
	} else {
		// Look for user-defined fields with pomdb tags at the root level
		typ := rv.Type()
		for j := 0; j < rv.NumField(); j++ {
			field := rv.Field(j)
			fieldType := typ.Field(j)
			tagValue := fieldType.Tag.Get("pomdb")

			if strings.Contains(tagValue, "id") {
				sc.IDField = &field
			}
			if strings.Contains(tagValue, "created_at") {
				sc.CreatedAtField = &field
			}
			if strings.Contains(tagValue, "updated_at") {
				sc.UpdatedAtField = &field
			}
			if strings.Contains(tagValue, "deleted_at") {
				sc.DeletedAtField = &field
			}
		}
	}

	return sc
}

func getFieldFromStruct(v reflect.Value, fieldName string) *reflect.Value {
	field := v.FieldByName(fieldName)
	if field.IsValid() {
		return &field
	}
	return nil
}
