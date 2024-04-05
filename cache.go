package pomdb

import (
	"fmt"
	"log"
	"reflect"

	"github.com/gertd/go-pluralize"
	"github.com/iancoleman/strcase"
)

type IndexField struct {
	FieldName     string
	CurrentValue  string
	PreviousValue string
	IsUnique      bool
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

		for j := 0; j < rv.NumField(); j++ {
			vpntr := rv.Field(j)
			value := rv.Field(j).String()
			fpntr := rv.Type().Field(j)
			pmtag := fpntr.Tag.Get("pomdb")
			jstag := fpntr.Tag.Get("json")

			// Set managed fields
			if tagContains(pmtag, []string{"id"}) {
				mc.ModelID = &vpntr
			}
			if tagContains(pmtag, []string{"created_at"}) {
				mc.CreatedAt = &vpntr
			}
			if tagContains(pmtag, []string{"updated_at"}) {
				mc.UpdatedAt = &vpntr
			}
			if tagContains(pmtag, []string{"deleted_at"}) {
				mc.DeletedAt = &vpntr
			}

			// Collect index fields
			if tagContains(pmtag, []string{"unique", "index"}) {
				mc.IndexFields = append(mc.IndexFields, IndexField{
					FieldName:    jstag,
					CurrentValue: value,
					IsUnique:     true,
				})
				continue
			}
			if tagContains(pmtag, []string{"index"}) {
				mc.IndexFields = append(mc.IndexFields, IndexField{
					FieldName:    jstag,
					CurrentValue: value,
					IsUnique:     false,
				})
			}
		}
	}

	return mc
}

// SetManagedFields sets the managed fields in the cache.
func (mc *ModelCache) SetManagedFields() {
	mc.ModelID.Set(reflect.ValueOf(NewULID()))

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
	return mc.ModelID.Interface().(ULID).String()
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
		value := fmt.Sprintf("%v", rv.MapIndex(reflect.ValueOf(index.FieldName)).Interface())

		if value != index.CurrentValue {
			mc.IndexFields[k].PreviousValue = value
			diff = true
		}
	}

	return diff
}
