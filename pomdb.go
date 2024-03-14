package pomdb

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type Client struct {
	Bucket  string
	Region  string
	Service *s3.Client
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
func (c *Client) CheckIndexExists(cache *ModelCache) error {
	for _, index := range cache.IndexFields {
		if index.IsUnique {
			// Encode the index field value in base64
			code := base64.StdEncoding.EncodeToString([]byte(index.CurrentValue))

			// Create the key path for the index item
			key := cache.Collection + "/indexes/" + index.FieldName + "/" + code

			head := &s3.HeadObjectInput{
				Bucket: &c.Bucket,
				Key:    &key,
			}

			var notFound *types.NotFound
			_, err := c.Service.HeadObject(context.TODO(), head)
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
func (c *Client) CreateIndexItems(cache *ModelCache) error {
	id := cache.ModelID.Interface().(ObjectID).String()

	for _, index := range cache.IndexFields {
		log.Printf("CreateIndexItem: collection=%s, indexField=%v", cache.Collection, index)

		// Encode the index field value in base64
		code := base64.StdEncoding.EncodeToString([]byte(index.CurrentValue))

		// Create the key path for the index item
		key := cache.Collection + "/indexes/" + index.FieldName + "/" + code

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
func (c *Client) UpdateIndexItems(cache *ModelCache) error {
	id := cache.ModelID.Interface().(ObjectID).String()

	for _, index := range cache.IndexFields {
		if index.PreviousValue != "" {
			log.Printf("UpdateIndexItem: collection=%s, indexField=%v", cache.Collection, index)

			// Encode the index field value in base64
			code := base64.StdEncoding.EncodeToString([]byte(index.PreviousValue))

			// Create the key path for the old index item
			oldKey := cache.Collection + "/indexes/" + index.FieldName + "/" + code

			// Delete the old index item
			del := &s3.DeleteObjectInput{
				Bucket: &c.Bucket,
				Key:    &oldKey,
			}

			if _, err := c.Service.DeleteObject(context.TODO(), del); err != nil {
				return err
			}

			// Encode the index field value in base64
			code = base64.StdEncoding.EncodeToString([]byte(index.CurrentValue))

			// Create the key path for the new index item
			newKey := cache.Collection + "/indexes/" + index.FieldName + "/" + code

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
