package pomdb

import (
	"context"
	"fmt"
	"log"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	Bucket  string
	Region  string
	Service *s3.Client
}

type Schema struct {
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

	c.Service = s3.NewFromConfig(conf)

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

	if _, err := c.Service.HeadBucket(context.TODO(), head); err != nil {
		return err
	}

	return nil
}

func (c *Collection) Create(model interface{}) error {
	if err := c.CheckModel(model); err != nil {
		return err
	}

	return nil
}

func (c *Collection) CheckModel(model interface{}) error {
	mtype := reflect.TypeOf(model)
	stype := reflect.TypeOf(c.Schema.Model)

	if mtype != stype {
		return fmt.Errorf(
			"model %s does not match schema %s",
			mtype.Name(),
			stype.Name(),
		)
	}

	return nil
}
