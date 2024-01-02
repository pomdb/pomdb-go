package pomdb

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type ObjectId [12]byte

func NewObjectId() ObjectId {
	var id ObjectId
	rand.Read(id[:]) // Replace with a more robust implementation.
	return id
}

func (id ObjectId) String() string {
	return fmt.Sprintf("%x", id[:])
}

type Client struct {
	Bucket  string
	Region  string
	Service *s3.Client
}

type Schema struct {
	Timestamps bool
}

type Collection[T any] struct {
	Client *Client
	Schema Schema
}

type Generic interface {
	Id() ObjectId
}

type Model[T any] struct {
	Client *Client
	Value  T
	Get    func() T
	Set    func(T)
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

func (c *Collection[T]) NewModel(v *T) *Model[T] {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Set Id (ObjectId)
	idField := val.FieldByName("Id")
	if idField.IsValid() && idField.Type() == reflect.TypeOf(ObjectId{}) {
		idField.Set(reflect.ValueOf(NewObjectId()))
	}

	// Set timestamps if required and fields exist
	if c.Schema.Timestamps {
		now := time.Now().Unix()
		setTimestamp(val, "CreatedAt", now)
		setTimestamp(val, "UpdatedAt", now)
	}

	return &Model[T]{
		Client: c.Client,
		Value:  *v,
	}
}

// Helper function to set timestamp fields.
func setTimestamp(val reflect.Value, fieldName string, timestamp int64) {
	field := val.FieldByName(fieldName)
	if field.IsValid() && field.Kind() == reflect.Int64 {
		field.SetInt(timestamp)
	}
}

func (m *Model[T]) Save() error {
	data, err := json.Marshal(m.Value)
	if err != nil {
		return err
	}

	val := reflect.ValueOf(m.Value)
	id := val.FieldByName("Id").String()

	input := &s3.PutObjectInput{
		Bucket: &m.Client.Bucket,
		Key:    aws.String(id),
		Body:   bytes.NewReader(data),
	}

	_, err = m.Client.Service.PutObject(context.TODO(), input)
	if err != nil {
		return err
	}

	return nil
}
