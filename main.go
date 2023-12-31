package pomdb

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	Bucket      string
	Region      string
	Schemas     map[string]*Schema
	Collections map[string]*Collection
	service     *s3.Client
}

type Schema struct {
	Name  string
	Model interface{}
}

type Collection struct {
	Client *Client
	Schema *Schema
}

func (c *Client) Connect() error {
	conf, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(c.Region),
	)
	if err != nil {
		return err
	}

	c.service = s3.NewFromConfig(conf)

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

	if _, err := c.service.HeadBucket(context.TODO(), head); err != nil {
		return err
	}

	return nil
}

func (c *Client) Collection(schema *Schema) *Collection {
	c.Schemas[schema.Name] = schema

	c.Collections[schema.Name] = &Collection{
		Client: c,
		Schema: schema,
	}

	return c.Collections[schema.Name]
}

func (s *Collection) Create(model interface{}) error {
	return nil
}
