package pomdb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Restore restores soft-deleted records and indexes in the database.
func (c *Client) Restore(i interface{}) (*string, error) {
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

	key := co + "/" + id

	// Restore the record
	del := &s3.DeleteObjectTaggingInput{
		Bucket: &c.Bucket,
		Key:    &key,
	}

	_, err = c.Service.DeleteObjectTagging(context.TODO(), del)
	if err != nil {
		return nil, err
	}

	return &id, nil
}
