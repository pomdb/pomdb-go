package pomdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

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
		pfx, err := encodeIndexPrefix(co, q.Field, q.Value, true)
		if err != nil {
			return nil, err
		}

		// Check if index exists
		lst := &s3.ListObjectsV2Input{
			Bucket: &c.Bucket,
			Prefix: &pfx,
		}

		res, err := c.Service.ListObjectsV2(context.TODO(), lst)
		if err != nil {
			return nil, err
		}

		if res.Contents == nil {
			return nil, fmt.Errorf("FindOne: index not found: collection=%s, field=%s, value=%s", co, q.Field, q.Value)
		}

		if len(res.Contents) > 1 {
			return nil, fmt.Errorf("FindOne: multiple records found: collection=%s, field=%s, value=%s", co, q.Field, q.Value)
		}

		// Get record id
		uid := strings.TrimPrefix(*res.Contents[0].Key, pfx+"/")

		// Set key path
		key = co + "/" + uid
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
				return nil, fmt.Errorf("FindOne: record not found: collection=%s, field=%s, value=%s", co, q.Field, q.Value)
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
		return nil, fmt.Errorf("FindOne: record not found: collection=%s, field=%s, value=%s", co, q.Field, q.Value)
	} else if err != nil {
		return nil, err
	}

	elem := reflect.TypeOf(q.Model).Elem()
	model := reflect.New(elem).Interface()
	err = json.NewDecoder(rec.Body).Decode(&model)
	if err != nil {
		return nil, err
	}

	return model, nil
}
