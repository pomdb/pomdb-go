package pomdb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Purge permanently removes a soft-deleted record and its indexes from the database.
func (c *Client) Purge(i interface{}) (*string, error) {
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

	// Check indexes
	if len(ca.IndexFields) > 0 {
		if err := c.DeleteIndexItems(ca); err != nil {
			return nil, err
		}
	}

	// Set the record's key
	key := co + "/" + id

	// Use s3 to delete the record
	del := &s3.DeleteObjectInput{
		Bucket: &c.Bucket,
		Key:    &key,
	}

	// Delete the record's data
	_, err = c.Service.DeleteObject(context.TODO(), del)
	if err != nil {
		return nil, err
	}

	return &id, nil
}
