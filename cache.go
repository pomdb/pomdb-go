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
	RangedIndex
)

type IndexField struct {
	FieldName     string
	FieldType     reflect.Type
	CurrentValue  string
	PreviousValue string
	IndexType     IndexType
}

type ModelCache struct {
	ModelID     *reflect.Value
	IndexFields []IndexField
	CreatedAt   *reflect.Value
	UpdatedAt   *reflect.Value
	DeletedAt   *reflect.Value
	Collection  string
	Reference   interface{}
}

func NewModelCache(rv reflect.Value) *ModelCache {
	mc := &ModelCache{}

	// Get the collection name
	mc.Collection = pluralize.NewClient().Plural(strcase.ToSnake(rv.Type().Name()))

	// Log the original and final names
	log.Printf("Collection: %s -> %s", rv.Type().Name(), mc.Collection)

	// Store a reference to the original struct
	mc.Reference = reflect.New(rv.Type()).Interface()

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
		field := rv.Field(j)
		fpntr := rv.Type().Field(j)
		pmtag := fpntr.Tag.Get("pomdb")
		jstag := fpntr.Tag.Get("json")

		value, err := stringifyFieldValue(field, fpntr)
		if err != nil {
			log.Printf("[Error] NewModelCache: %v", err)
			continue
		}

		indexField := IndexField{
			FieldName:    jstag,
			FieldType:    fpntr.Type,
			CurrentValue: value,
		}

		if tagContains(pmtag, []string{"ranged", "index"}) {
			indexField.IndexType = RangedIndex
			mc.IndexFields = append(mc.IndexFields, indexField)
			continue
		}
		if tagContains(pmtag, []string{"unique", "index"}) {
			indexField.IndexType = UniqueIndex
			mc.IndexFields = append(mc.IndexFields, indexField)
			continue
		}
		if tagContains(pmtag, []string{"index"}) {
			indexField.IndexType = SharedIndex
			mc.IndexFields = append(mc.IndexFields, indexField)
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
func (mc *ModelCache) CompareIndexFields(model interface{}) bool {
	modval := reflect.ValueOf(model).Elem()
	modtyp := modval.Type()

	tagmap := make(map[string]string)
	for i := 0; i < modtyp.NumField(); i++ {
		field := modtyp.Field(i)
		jstag := field.Tag.Get("json")
		if jstag != "" {
			tagmap[jstag] = field.Name
		}
	}

	diff := false
	for k, index := range mc.IndexFields {
		fldnme, ok := tagmap[index.FieldName]
		if !ok {
			log.Printf("json tag %s not found in model", index.FieldName)
			continue
		}

		fldval := modval.FieldByName(fldnme)
		if !fldval.IsValid() {
			log.Printf("field %s not found in model", fldnme)
			continue
		}

		var newval string
		if fldval.Type().AssignableTo(reflect.TypeOf(Timestamp{})) {
			newval = fldval.Interface().(Timestamp).String()
		} else {
			newval = fmt.Sprintf("%v", fldval.Interface())
		}

		if newval != index.CurrentValue {
			mc.IndexFields[k].PreviousValue = newval
			diff = true
		}
	}

	return diff
}
