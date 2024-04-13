package pomdb

import (
	"context"
	"errors"
	"fmt"
	"log"

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
	col := ca.Collection
	for _, idx := range ca.IndexFields {
		if idx.IndexType == UniqueIndex {
			// Create the pfx path for the index item
			pfx, err := encodeIndexPrefix(col, idx.CurrValues, idx.IndexType)
			if err != nil {
				return err
			}

			list := &s3.ListObjectsV2Input{
				Bucket: &c.Bucket,
				Prefix: &pfx,
			}

			res, err := c.Service.ListObjectsV2(context.TODO(), list)
			if err != nil {
				return err
			}

			if res.Contents == nil {
				return nil
			}

			if len(res.Contents) > 0 {
				return fmt.Errorf("[Error] CheckIndexExists: index %s with value %s already exists", idx.IndexName, idx.CurrValues[idx.IndexName])
			}
		}
	}

	return nil
}

// CreateIndexItems creates an index item in the given collection.
func (c *Client) CreateIndexItems(ca *ModelCache) error {
	uid := ca.GetModelID()
	col := ca.Collection

	for _, idx := range ca.IndexFields {
		log.Printf("CreateIndexItem: collection=%s, indexField=%v", col, idx)

		// Create the pfx path for the index item
		pfx, err := encodeIndexPrefix(col, idx.CurrValues, idx.IndexType)
		if err != nil {
			return err
		}

		put := &s3.PutObjectInput{
			Bucket: &c.Bucket,
			Key:    aws.String(pfx + "/" + uid),
		}

		if _, err := c.Service.PutObject(context.TODO(), put); err != nil {
			return err
		}
	}

	return nil
}

// UpdateIndexItems updates index items in the given collection.
func (c *Client) UpdateIndexItems(ca *ModelCache) error {
	uid := ca.GetModelID()
	col := ca.Collection

	for _, idx := range ca.IndexFields {
		if len(idx.PrevValues) > 0 {
			log.Printf("UpdateIndexItem: collection=%s, indexField=%v", col, idx)

			// Create the key path for the old index item
			oldPfx, err := encodeIndexPrefix(col, idx.PrevValues, idx.IndexType)
			if err != nil {
				return err
			}

			// Delete the old index item
			del := &s3.DeleteObjectInput{
				Bucket: &c.Bucket,
				Key:    aws.String(oldPfx + "/" + uid),
			}

			if _, err := c.Service.DeleteObject(context.TODO(), del); err != nil {
				return err
			}

			// Create the key path for the new index item
			newPfx, err := encodeIndexPrefix(col, idx.CurrValues, idx.IndexType)
			if err != nil {
				return err
			}

			put := &s3.PutObjectInput{
				Bucket: &c.Bucket,
				Key:    aws.String(newPfx + "/" + uid),
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
	uid := ca.GetModelID()
	col := ca.Collection

	for _, idx := range ca.IndexFields {
		log.Printf("DeleteIndexItem: collection=%s, indexField=%v", col, idx)

		// Create the pfx path for the index item
		pfx, err := encodeIndexPrefix(col, idx.CurrValues, idx.IndexType)
		if err != nil {
			return err
		}

		del := &s3.DeleteObjectInput{
			Bucket: &c.Bucket,
			Key:    aws.String(pfx + "/" + uid),
		}

		var notFound *types.NotFound
		_, err = c.Service.DeleteObject(context.TODO(), del)
		if err != nil && !errors.As(err, &notFound) {
			return err
		}
	}

	return nil
}
