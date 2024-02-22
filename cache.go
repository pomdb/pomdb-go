package pomdb

import (
	"log"
	"reflect"
	"strings"

	"github.com/gertd/go-pluralize"
	"github.com/iancoleman/strcase"
)

type ModelCache struct {
	ModelID    *reflect.Value
	CreatedAt  *reflect.Value
	UpdatedAt  *reflect.Value
	DeletedAt  *reflect.Value
	Collection string
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

	return mc
}

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

func (mc *ModelCache) SetUpdatedAt() {
	if mc.UpdatedAt != nil && mc.UpdatedAt.CanSet() {
		mc.UpdatedAt.Set(reflect.ValueOf(NewTimestamp()))
	}
}

func (mc *ModelCache) SetDeletedAt() {
	if mc.DeletedAt != nil && mc.DeletedAt.CanSet() {
		mc.DeletedAt.Set(reflect.ValueOf(NilTimestamp()))
	}
}
