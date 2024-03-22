package pomdb

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Delete deletes a record and its indexes from the database.
func (c *Client) Delete(i interface{}) (*string, error) {
	if c.SoftDeletes {
		return c.SoftDelete(i)
	}

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

// SoftDelete soft deletes a record and its indexes from the database.
func (c *Client) SoftDelete(i interface{}) (*string, error) {
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

	// Create a deletion timestamp in seconds
	ts := strconv.FormatInt(time.Now().Unix(), 10)

	// Add the `DeletedAt` tag to the record
	put := &s3.PutObjectTaggingInput{
		Bucket: &c.Bucket,
		Key:    &key,
		Tagging: &types.Tagging{
			TagSet: []types.Tag{
				{
					Key:   aws.String("DeletedAt"),
					Value: aws.String(ts),
				},
			},
		},
	}

	if _, err := c.Service.PutObjectTagging(context.TODO(), put); err != nil {
		return nil, err
	}

	// Check indexes
	if len(ca.IndexFields) > 0 {
		if err := c.SoftDeleteIndexItems(ca); err != nil {
			return nil, err
		}
	}

	return &id, nil
}
