package pomdb

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Update updates a record in the database.
func (c *Client) Update(i interface{}) (*string, error) {
	// Dereference the input
	rv, err := dereferenceStruct(i)
	if err != nil {
		return nil, err
	}

	// Build the struct cache
	ca := NewModelCache(rv)

	// Get the model ID
	id := ca.GetModelID()

	// Get the collection
	co := ca.Collection

	// Set the record's key
	key := co + "/" + id

	// Use s3 to get the record
	get := &s3.GetObjectInput{
		Bucket: &c.Bucket,
		Key:    &key,
	}

	// Get the record's data
	doc, err := c.Service.GetObject(context.TODO(), get)
	if err != nil {
		return nil, err
	}

	// Unmarshal the record
	elem := reflect.TypeOf(ca.Reference).Elem()
	model := reflect.New(elem).Interface()
	if err := json.NewDecoder(doc.Body).Decode(&model); err != nil {
		return nil, err
	}

	// Check/update indexes
	if len(ca.IndexFields) > 0 {
		if diff := ca.CompareIndexFields(model); diff {
			if err := c.CheckIndexExists(ca); err != nil {
				return nil, err
			}

			if err := c.UpdateIndexItems(ca); err != nil {
				return nil, err
			}
		}
	}

	// Encode the object
	enc, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	// Set the record's data
	put := &s3.PutObjectInput{
		Bucket: &c.Bucket,
		Key:    &key,
		Body:   bytes.NewReader(enc),
	}

	// Set the record's etag
	res, err := c.Service.PutObject(context.TODO(), put)
	if err != nil {
		return nil, err
	}

	return res.ETag, nil
}
