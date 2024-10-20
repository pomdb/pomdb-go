package pomdb

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Create creates a record in the database
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

	if len(ca.IndexFields) > 0 {
		if err := c.CheckIndexExists(ca); err != nil {
			return nil, err
		}

		if err := c.CreateIndexItems(ca); err != nil {
			return nil, err
		}
	}

	// Encode the object
	enc, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	// Set the record's key
	key := co + "/" + id

	put := &s3.PutObjectInput{
		Bucket: &c.Bucket,
		Key:    &key,
		Body:   bytes.NewReader(enc),
	}

	// Set the record's data
	res, err := c.Service.PutObject(context.TODO(), put)
	if err != nil {
		return nil, err
	}

	return res.ETag, nil
}
