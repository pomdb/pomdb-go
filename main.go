package pomdb

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type ObjectID [12]byte

type Statement struct {
	Model interface{}
}

type Client struct {
	Bucket    string
	Region    string
	Service   *s3.Client
	Statement *Statement
}

type Model struct {
	ID        ObjectID
	CreatedAt int64
	UpdatedAt int64
	DeletedAt int64
}

func NewObjectID() ObjectID {
	var id ObjectID
	rand.Read(id[:]) // Replace with a more robust implementation.
	return id
}

func (id ObjectID) String() string {
	return fmt.Sprintf("%x", id[:])
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

func (c *Client) Create(model interface{}) error {
	val := reflect.ValueOf(model)

	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("model must be a pointer to a struct")
	}

	if _, ok := val.Elem().FieldByName("ID").Interface().(ObjectID); !ok {
		return fmt.Errorf("model must have ID field of type ObjectID")
	}

	val.Elem().FieldByName("ID").Set(reflect.ValueOf(NewObjectID()))

	return nil
}
