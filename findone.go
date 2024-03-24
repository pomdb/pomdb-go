package pomdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// FindOne finds a single record in the database. It returns an error if the
// record is not found.
func (c *Client) FindOne(q Query) (interface{}, error) {
	target := "record"
	if q.Field != "id" {
		target = "index"
	}

	// Dereference q.Model
	rv, err := dereferenceStruct(q.Model)
	if err != nil {
		return nil, err
	}

	// Build the struct cache
	ca := NewModelCache(rv)

	// Get the collection
	co := ca.Collection

	// Set record key path
	key := co + "/" + q.Value

	if target == "index" {
		// Set index key path
		key, err = encodeIndexKey(co, q.Field, q.Value)
		if err != nil {
			return nil, err
		}
	}

	// Filter soft deletes
	if c.SoftDeletes {
		tag := &s3.GetObjectTaggingInput{
			Bucket: &c.Bucket,
			Key:    &key,
		}

		tags, err := c.Service.GetObjectTagging(context.TODO(), tag)
		if err != nil {
			return nil, err
		}

		for _, t := range tags.TagSet {
			if *t.Key == "DeletedAt" {
				return nil, fmt.Errorf("FindOne: %s not found: collection=%s, field=%s, value=%s", target, co, q.Field, q.Value)
			}
		}
	}

	get := &s3.GetObjectInput{
		Bucket: &c.Bucket,
		Key:    &key,
	}

	// Fetch the record
	var noSuchKey *types.NoSuchKey
	rec, err := c.Service.GetObject(context.TODO(), get)
	if err != nil && errors.As(err, &noSuchKey) {
		return nil, fmt.Errorf("FindOne: %s not found: collection=%s, field=%s, value=%s", target, co, q.Field, q.Value)
	} else if err != nil {
		return nil, err
	}

	if target == "index" {
		bdy, err := io.ReadAll(rec.Body)
		if err != nil {
			return nil, err
		}

		id := string(bdy)

		key = co + "/" + id

		get = &s3.GetObjectInput{
			Bucket: &c.Bucket,
			Key:    &key,
		}

		// Fetch the record
		rec, err = c.Service.GetObject(context.TODO(), get)
		if err != nil && errors.As(err, &noSuchKey) {
			return nil, fmt.Errorf("FindOne: record not found: collection=%s, id=%s", co, id)
		} else if err != nil {
			return nil, err
		}
	}

	elem := reflect.TypeOf(q.Model).Elem()
	model := reflect.New(elem).Interface()
	err = json.NewDecoder(rec.Body).Decode(&model)
	if err != nil {
		return nil, err
	}

	return model, nil
}
