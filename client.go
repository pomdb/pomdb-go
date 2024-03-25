package pomdb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type Client struct {
	Service     *s3.Client
	Bucket      string
	Region      string
	SoftDeletes bool
	Pessimistic bool
	Optimistic  bool
}

func (c *Client) Connect() error {
	conf, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(c.Region),
	)
	if err != nil {
		return err
	}

	c.Service = s3.NewFromConfig(conf)

	if err := c.CheckBucket(); err != nil {
		return fmt.Errorf("bucket %s does not exist", c.Bucket)
	}

	log.Printf("connected to %s", c.Bucket)

	return nil
}

func (c *Client) CheckBucket() error {
	head := &s3.HeadBucketInput{
		Bucket: &c.Bucket,
	}

	if _, err := c.Service.HeadBucket(context.TODO(), head); err != nil {
		return err
	}

	return nil
}

// CheckIndexExists checks if an index item exists in the given collection.
func (c *Client) CheckIndexExists(ca *ModelCache) error {
	for _, index := range ca.IndexFields {
		if index.IsUnique {
			// Create the key path for the index item
			key, err := encodeIndexKey(ca.Collection, index.FieldName, index.CurrentValue)
			if err != nil {
				return err
			}

			head := &s3.HeadObjectInput{
				Bucket: &c.Bucket,
				Key:    &key,
			}

			var notFound *types.NotFound
			_, err = c.Service.HeadObject(context.TODO(), head)
			if err != nil && !errors.As(err, &notFound) {
				return err
			}

			if err == nil {
				return fmt.Errorf("[Error] CheckIndexExists: index %s with value %s already exists", index.FieldName, index.CurrentValue)
			}
		}
	}

	return nil
}

// CreateIndexItems creates an index item in the given collection.
func (c *Client) CreateIndexItems(ca *ModelCache) error {
	id := ca.ModelID.Interface().(ULID).String()

	for _, index := range ca.IndexFields {
		log.Printf("CreateIndexItem: collection=%s, indexField=%v", ca.Collection, index)

		// Create the key path for the index item
		key, err := encodeIndexKey(ca.Collection, index.FieldName, index.CurrentValue)
		if err != nil {
			return err
		}

		put := &s3.PutObjectInput{
			Bucket: &c.Bucket,
			Key:    &key,
			Body:   bytes.NewReader([]byte(id)),
		}

		if _, err := c.Service.PutObject(context.TODO(), put); err != nil {
			return err
		}
	}

	return nil
}

// UpdateIndexItems updates index items in the given collection.
func (c *Client) UpdateIndexItems(ca *ModelCache) error {
	id := ca.ModelID.Interface().(ULID).String()

	for _, index := range ca.IndexFields {
		if index.PreviousValue != "" {
			log.Printf("UpdateIndexItem: collection=%s, indexField=%v", ca.Collection, index)

			// Create the key path for the old index item
			oldKey, err := encodeIndexKey(ca.Collection, index.FieldName, index.PreviousValue)
			if err != nil {
				return err
			}

			// Delete the old index item
			del := &s3.DeleteObjectInput{
				Bucket: &c.Bucket,
				Key:    &oldKey,
			}

			if _, err := c.Service.DeleteObject(context.TODO(), del); err != nil {
				return err
			}

			// Create the key path for the new index item
			newKey, err := encodeIndexKey(ca.Collection, index.FieldName, index.CurrentValue)
			if err != nil {
				return err
			}

			put := &s3.PutObjectInput{
				Bucket: &c.Bucket,
				Key:    &newKey,
				Body:   bytes.NewReader([]byte(id)),
			}

			if _, err := c.Service.PutObject(context.TODO(), put); err != nil {
				return err
			}
		}
	}

	return nil
}

// DeleteIndexItems deletes index items in the given collection.
func (c *Client) DeleteIndexItems(ca *ModelCache) error {
	for _, index := range ca.IndexFields {
		log.Printf("DeleteIndexItem: collection=%s, indexField=%v", ca.Collection, index)

		// Create the key path for the index item
		key, err := encodeIndexKey(ca.Collection, index.FieldName, index.CurrentValue)
		if err != nil {
			return err
		}

		del := &s3.DeleteObjectInput{
			Bucket: &c.Bucket,
			Key:    &key,
		}

		var notFound *types.NotFound
		_, err = c.Service.DeleteObject(context.TODO(), del)
		if err != nil && !errors.As(err, &notFound) {
			return err
		}
	}

	return nil
}

// SoftDeleteIndexItems adds a `DeletedAt` tag to the record's indexes.
func (c *Client) SoftDeleteIndexItems(ca *ModelCache) error {
	for _, index := range ca.IndexFields {
		log.Printf("SoftDeleteIndexItem: collection=%s, indexField=%v", ca.Collection, index)

		// Create the key path for the index item
		key, err := encodeIndexKey(ca.Collection, index.FieldName, index.CurrentValue)
		if err != nil {
			return err
		}

		// Create a deletion timestamp in seconds
		ts := strconv.FormatInt(time.Now().Unix(), 10)

		// Add the `DeletedAt` tag to the index item
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
			return err
		}
	}

	return nil
}

// RestoreIndexItems removes the `DeletedAt` tag from the record's indexes.
func (c *Client) RestoreIndexItems(ca *ModelCache) error {
	for _, index := range ca.IndexFields {
		log.Printf("RestoreIndexItem: collection=%s, indexField=%v", ca.Collection, index)

		// Create the key path for the index item
		key, err := encodeIndexKey(ca.Collection, index.FieldName, index.CurrentValue)
		if err != nil {
			return err
		}

		// Remove the `DeletedAt` tag from the index item
		del := &s3.DeleteObjectTaggingInput{
			Bucket: &c.Bucket,
			Key:    &key,
		}

		if _, err := c.Service.DeleteObjectTagging(context.TODO(), del); err != nil {
			return err
		}
	}

	return nil
}

// PurgeIndexItems permanently removes the record's indexes from the database.
func (c *Client) PurgeIndexItems(ca *ModelCache) error {
	for _, index := range ca.IndexFields {
		log.Printf("PurgeIndexItem: collection=%s, indexField=%v", ca.Collection, index)

		// Create the key path for the index item
		key, err := encodeIndexKey(ca.Collection, index.FieldName, index.CurrentValue)
		if err != nil {
			return err
		}

		del := &s3.DeleteObjectInput{
			Bucket: &c.Bucket,
			Key:    &key,
		}

		if _, err := c.Service.DeleteObject(context.TODO(), del); err != nil {
			return err
		}
	}

	return nil
}
