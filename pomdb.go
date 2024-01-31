package pomdb

import (
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
func (c *Client) CheckIndexExists(collection string, indexFields []IndexFieldValue) error {
	for _, indexField := range indexFields {
		log.Printf("CheckIndexExists: collection=%s, indexField=%v", collection, indexField)

		// Encode the index field value in base64
		code := base64.StdEncoding.EncodeToString([]byte(indexField.Value))

		// Create the key path for the index item
		key := collection + "/indexes/" + indexField.Field + "/" + code

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
			return fmt.Errorf("index item already exists")
		}
	}

	return nil
}

// CreateIndexItem creates an index item in the given collection.
func (c *Client) CreateIndexItem(collection string, indexFields []IndexFieldValue) error {
	for _, indexField := range indexFields {
		log.Printf("CreateIndexItem: collection=%s, indexField=%v", collection, indexField)

		// Encode the index field value in base64
		code := base64.StdEncoding.EncodeToString([]byte(indexField.Value))

		// Create the key path for the index item
		key := collection + "/indexes/" + indexField.Field + "/" + code

		put := &s3.PutObjectInput{
			Bucket: &c.Bucket,
			Key:    &key,
		}

		if _, err := c.Service.PutObject(context.TODO(), put); err != nil {
			return err
		}
	}

	return nil
}
