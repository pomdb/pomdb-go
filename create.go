package pomdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/antchfx/jsonquery"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (c *Client) Create(i interface{}) error {
	rv := reflect.ValueOf(i)

	co := GetCollectionName(i)

	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("model must be a pointer to a struct")
	}

	if _, ok := rv.Elem().FieldByName("ID").Interface().(ObjectID); !ok {
		return fmt.Errorf("model must have ID field of type ObjectID")
	}

	// Create a var to hold a reference to the unqiue field
	var uniqueField string
	var uniqueValue string

	for j := 0; j < rv.Elem().NumField(); j++ {
		field := rv.Elem().Type().Field(j)
		value := rv.Elem().Field(j).String()
		if strings.Contains(field.Tag.Get("pomdb"), "unique") {
			tagname := field.Tag.Get("json")

			log.Printf("model has unique field: %s", tagname)

			uniqueField = tagname
			uniqueValue = value
		}
	}

	id := NewObjectID()

	rv.Elem().FieldByName("ID").Set(reflect.ValueOf(id))

	if field := rv.Elem().FieldByName("CreatedAt"); field.IsValid() {
		field.SetInt(time.Now().Unix())
	}

	if field := rv.Elem().FieldByName("UpdatedAt"); field.IsValid() {
		field.SetInt(time.Now().Unix())
	}

	if field := rv.Elem().FieldByName("DeletedAt"); field.IsValid() {
		field.SetInt(0)
	}

	obj, err := json.Marshal(i)
	if err != nil {
		return err
	}

	var record []byte

	// Create input for HeadObject
	head := &s3.HeadObjectInput{
		Bucket: &c.Bucket,
		Key:    aws.String(co + ".json"),
	}

	if _, err := c.Service.HeadObject(context.TODO(), head); err == nil {
		// Create input for GetObject
		get := &s3.GetObjectInput{
			Bucket: &c.Bucket,
			Key:    aws.String(co + ".json"),
		}

		// Fetch the object from S3
		res, err := c.Service.GetObject(context.TODO(), get)
		if err != nil {
			return err
		}

		// Put contents of object into record
		record, err = io.ReadAll(res.Body)
		if err != nil {
			return err
		}

		// Close the response body
		res.Body.Close()
	}

	// Read the record line by line and check if the unique field matches the value
	// of the unique field in the model
	for _, line := range strings.Split(string(record), "\n") {
		if line == "" {
			continue
		}

		doc, err := jsonquery.Parse(strings.NewReader(line))
		if err != nil {
			return fmt.Errorf("failed to parse json: %s", err)
		}

		prop := jsonquery.FindOne(doc, uniqueField)
		if prop == nil {
			continue
		}

		if prop.Value() == uniqueValue {
			return fmt.Errorf("%s %s already exists", uniqueField, uniqueValue)
		}
	}

	// Append newline to object
	obj = append(obj, []byte("\n")...)

	// Append object to record
	record = append(record, obj...)

	// Create input for PutObject
	put := &s3.PutObjectInput{
		Bucket: &c.Bucket,
		Key:    aws.String(co + ".json"),
		Body:   strings.NewReader(string(record)),
	}

	// Put the object into S3
	if _, err := c.Service.PutObject(context.TODO(), put); err != nil {
		return err
	}

	return nil
}
