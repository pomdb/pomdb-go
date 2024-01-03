package pomdb

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	Bucket string
	Region string
	S3     *s3.Client
}

func (c *Client) Connect() error {
	conf, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(c.Region),
	)
	if err != nil {
		return err
	}

	c.S3 = s3.NewFromConfig(conf)

	if err := c.CheckBucket(); err != nil {
		return err
	}

	log.Printf("Connected to %s", c.Bucket)

	return nil
}

func (c *Client) CheckBucket() error {
	head := &s3.HeadBucketInput{
		Bucket: &c.Bucket,
	}

	if _, err := c.S3.HeadBucket(context.TODO(), head); err != nil {
		return err
	}

	return nil
}
