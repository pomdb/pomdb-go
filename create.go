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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func (c *Client) Create(i interface{}) error {
	rv := reflect.ValueOf(i)

	co := getCollectionName(i)

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
		// Query the collection for the unique field using s3 select
		query := fmt.Sprintf("SELECT * FROM S3Object s WHERE s.%s = '%s'", uniqueField, uniqueValue)

		// Create input for SelectObjectContent
		selectInput := &s3.SelectObjectContentInput{
			Bucket:         &c.Bucket,
			Key:            aws.String(co + ".json"),
			Expression:     aws.String(query),
			ExpressionType: types.ExpressionTypeSql,
			InputSerialization: &types.InputSerialization{
				JSON: &types.JSONInput{
					Type: types.JSONTypeDocument,
				},
			},
			OutputSerialization: &types.OutputSerialization{
				JSON: &types.JSONOutput{
					RecordDelimiter: aws.String("\n"),
				},
			},
		}

		// Execute the query
		sel, err := c.Service.SelectObjectContent(context.TODO(), selectInput)
		if err != nil {
			return fmt.Errorf("failed to execute query: %s", err)
		}

		// Read the results
		for event := range sel.GetStream().Events() {
			switch event := event.(type) {
			case *types.SelectObjectContentEventStreamMemberRecords:
				record = event.Value.Payload
			}

			if record != nil {
				log.Printf("found record: %s", record)
				break
			}
		}

		// Close the response body
		sel.GetStream().Close()

		// If the record is not nil, then the unique field already exists
		if record != nil {
			return fmt.Errorf("unique field %s already exists", uniqueField)
		}

		// Create input for GetObject
		getInput := &s3.GetObjectInput{
			Bucket: &c.Bucket,
			Key:    aws.String(co + ".json"),
		}

		// Fetch the object from S3
		get, err := c.Service.GetObject(context.TODO(), getInput)
		if err != nil {
			return err
		}

		// Put contents of object into record
		record, err = io.ReadAll(get.Body)
		if err != nil {
			return err
		}

		// Close the response body
		get.Body.Close()
	}

	// Append newline to object
	obj = append(obj, []byte("\n")...)

	// Append object to record
	record = append(record, obj...)

	// Create input for PutObject
	putInput := &s3.PutObjectInput{
		Bucket: &c.Bucket,
		Key:    aws.String(co + ".json"),
		Body:   strings.NewReader(string(record)),
	}

	// Put the object into S3
	if _, err := c.Service.PutObject(context.TODO(), putInput); err != nil {
		return err
	}

	return nil
}
