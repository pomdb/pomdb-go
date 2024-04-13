package pomdb

import (
	"fmt"
	"log"
	"reflect"

	"github.com/gertd/go-pluralize"
	"github.com/iancoleman/strcase"
)

type IndexType int

const (
	UniqueIndex IndexType = iota
	SharedIndex
	CompositeIndex
)

type IndexField struct {
	IndexName  string
	CurrValues map[string]string
	PrevValues map[string]string
	IndexType  IndexType
}

type ModelCache struct {
	ModelID     *reflect.Value
	IndexFields map[string]IndexField
	CreatedAt   *reflect.Value
	UpdatedAt   *reflect.Value
	DeletedAt   *reflect.Value
	Collection  string
}

func NewModelCache(rv reflect.Value) *ModelCache {
	mc := &ModelCache{
		IndexFields: make(map[string]IndexField),
	}

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
			fpntr := rv.Type().Field(j)
			pmtag := fpntr.Tag.Get("pomdb")

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
		}
	}

	for j := 0; j < rv.NumField(); j++ {
		value := rv.Field(j).String()
		fpntr := rv.Type().Field(j)
		pmtag := fpntr.Tag.Get("pomdb")
		jstag := fpntr.Tag.Get("json")

		// Collect index fields
		if tagContains(pmtag, []string{"composite", "index"}) {
			name := getCompositeIndexName(pmtag)

			if _, ok := mc.IndexFields[name]; !ok {
				mc.IndexFields[name] = IndexField{
					IndexName:  name,
					CurrValues: map[string]string{jstag: value},
					IndexType:  CompositeIndex,
				}
			} else {
				mc.IndexFields[name].CurrValues[jstag] = value
			}

			continue
		}
		if tagContains(pmtag, []string{"unique", "index"}) {
			mc.IndexFields[jstag] = IndexField{
				IndexName:  jstag,
				CurrValues: map[string]string{jstag: value},
				IndexType:  UniqueIndex,
			}

			continue
		}
		if tagContains(pmtag, []string{"index"}) {
			mc.IndexFields[jstag] = IndexField{
				IndexName:  jstag,
				CurrValues: map[string]string{jstag: value},
				IndexType:  SharedIndex,
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

// CompareIndexFields compares the index fields in the cache to the input,
// where 'i' is expected to be a map or a struct representing the previous
// state
func (mc *ModelCache) CompareIndexFields(i interface{}) bool {
	rv := reflect.ValueOf(i) // represents a map
	diff := false

	for k, index := range mc.IndexFields {
		if index.IndexType == CompositeIndex {
			for field, value := range index.CurrValues {
				if value != fmt.Sprintf("%v", rv.MapIndex(reflect.ValueOf(field)).Interface()) {
					mc.IndexFields[k].PrevValues[field] = value
				}
			}
			continue
		}

		value := fmt.Sprintf("%v", rv.MapIndex(reflect.ValueOf(index.IndexName)).Interface())
		if value != index.CurrValues[index.IndexName] {
			mc.IndexFields[k].PrevValues[index.IndexName] = index.CurrValues[index.IndexName]
		}
	}

	for _, index := range mc.IndexFields {
		if len(index.PrevValues) > 0 {
			diff = true
			break
		}
	}

	return diff
}
