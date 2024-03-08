package pomdb

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/gertd/go-pluralize"
	"github.com/iancoleman/strcase"
)

type IndexField struct {
	Field string
	Value string
	OlVal string
}

type ModelCache struct {
	ModelID     *reflect.Value
	IndexFields []IndexField
	CreatedAt   *reflect.Value
	UpdatedAt   *reflect.Value
	DeletedAt   *reflect.Value
	Collection  string
}

func NewModelCache(rv reflect.Value) *ModelCache {
	mc := &ModelCache{}

	// Get the collection name
	mc.Collection = pluralize.NewClient().Plural(strcase.ToSnake(rv.Type().Name()))

	// Log the original and final names
	log.Printf("Collection: %s -> %s", rv.Type().Name(), mc.Collection)

	if hasPomdbModel(rv) {
		// Use fields from embedded pomdb.Model
		mc.ModelID = getFieldByName(rv, "ID")
		mc.CreatedAt = getFieldByName(rv, "CreatedAt")
		mc.UpdatedAt = getFieldByName(rv, "UpdatedAt")
		mc.DeletedAt = getFieldByName(rv, "DeletedAt")
	} else {
		// Look for user-defined fields with pomdb tags
		typ := rv.Type()
		for j := 0; j < rv.NumField(); j++ {
			field := rv.Field(j)
			fieldType := typ.Field(j)
			tagValue := fieldType.Tag.Get("pomdb")

			if strings.Contains(tagValue, "id") {
				mc.ModelID = &field
			}
			if strings.Contains(tagValue, "created_at") {
				mc.CreatedAt = &field
			}
			if strings.Contains(tagValue, "updated_at") {
				mc.UpdatedAt = &field
			}
			if strings.Contains(tagValue, "deleted_at") {
				mc.DeletedAt = &field
			}
		}
	}

	for j := 0; j < rv.NumField(); j++ {
		field := rv.Type().Field(j)
		value := rv.Field(j).String()
		if strings.Contains(field.Tag.Get("pomdb"), "index") {
			tagname := field.Tag.Get("json")

			log.Printf("model has unique field: %s", tagname)

			mc.IndexFields = append(mc.IndexFields, IndexField{
				Field: tagname,
				Value: value,
			})
		}
	}

	return mc
}

// SetManagedFields sets the managed fields in the cache.
func (mc *ModelCache) SetManagedFields() {
	mc.ModelID.Set(reflect.ValueOf(NewObjectID()))

	if mc.CreatedAt != nil && mc.CreatedAt.CanSet() {
		mc.CreatedAt.Set(reflect.ValueOf(NewTimestamp()))
	}
	if mc.UpdatedAt != nil && mc.UpdatedAt.CanSet() {
		mc.UpdatedAt.Set(reflect.ValueOf(NewTimestamp()))
	}
	if mc.DeletedAt != nil && mc.DeletedAt.CanSet() {
		mc.DeletedAt.Set(reflect.ValueOf(NilTimestamp()))
	}
}

// GetModelID returns the model ID from the cache.
func (mc *ModelCache) GetModelID() string {
	return mc.ModelID.Interface().(ObjectID).String()
}

// SetUpdatedAt sets the UpdatedAt field in the cache.
func (mc *ModelCache) SetUpdatedAt() {
	if mc.UpdatedAt != nil && mc.UpdatedAt.CanSet() {
		mc.UpdatedAt.Set(reflect.ValueOf(NewTimestamp()))
	}
}

// SetDeletedAt sets the DeletedAt field in the cache.
func (mc *ModelCache) SetDeletedAt() {
	if mc.DeletedAt != nil && mc.DeletedAt.CanSet() {
		mc.DeletedAt.Set(reflect.ValueOf(NilTimestamp()))
	}
}

// CompareIndexFields compares the index fields in the cache to the input.
func (mc *ModelCache) CompareIndexFields(i interface{}) bool {
	rv := reflect.ValueOf(i) // represents a map

	diff := false
	for k, index := range mc.IndexFields {
		value := fmt.Sprintf("%v", rv.MapIndex(reflect.ValueOf(index.Field)).Interface())

		if value != index.Value {
			mc.IndexFields[k].OlVal = value
			diff = true
		}
	}

	return diff
}
