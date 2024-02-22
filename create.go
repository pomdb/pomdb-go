package pomdb

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (c *Client) Create(i interface{}) (*string, error) {
	// Dereference the input
	rv, err := dereferenceStruct(i)
	if err != nil {
		return nil, err
	}

	// Build the struct cache
	ca := NewModelCache(rv)

	// Set the new model fields
	ca.SetManagedFields()

	// Get the model ID
	id := ca.GetModelID()

	// Get the collection
	co := ca.Collection

	if ifv := getIndexFieldValues(rv, id); len(ifv) > 0 {
		if err := c.CheckIndexExists(co, ifv); err != nil {
			return nil, err
		}

		if err := c.CreateIndexItem(co, ifv); err != nil {
			return nil, err
		}
	}

	// Marshal the record
	rec, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	// Set the record's key
	key := co + "/" + id

	put := &s3.PutObjectInput{
		Bucket: &c.Bucket,
		Key:    &key,
		Body:   bytes.NewReader(rec),
	}

	// Set the record's data
	res, err := c.Service.PutObject(context.TODO(), put)
	if err != nil {
		return nil, err
	}

	return res.ETag, nil
}
