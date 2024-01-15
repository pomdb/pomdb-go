package pomdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func (c *Client) Create(i interface{}) error {
	rv := reflect.ValueOf(i)

	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("model must be a pointer to a struct")
	}

	if _, ok := rv.Elem().FieldByName("ID").Interface().(ObjectID); !ok {
		return fmt.Errorf("model must have ID field of type ObjectID")
	}

	rv.Elem().FieldByName("ID").Set(reflect.ValueOf(NewObjectID()))

	if field := rv.Elem().FieldByName("CreatedAt"); field.IsValid() {
		field.SetInt(time.Now().Unix())
	}

	if field := rv.Elem().FieldByName("UpdatedAt"); field.IsValid() {
		field.SetInt(time.Now().Unix())
	}

	if field := rv.Elem().FieldByName("DeletedAt"); field.IsValid() {
		field.SetInt(0)
	}

	co := getCollectionName(i)

	if uf, uv := getUniqueFieldMeta(rv); uf != "" {
		// Create query for SelectObjectContent
		query := fmt.Sprintf("SELECT * FROM S3Object s WHERE s.%s = '%s'", uf, uv)

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
			var noSuchKey *types.NoSuchKey
			if errors.As(err, &noSuchKey) {
				return fmt.Errorf("collection %s does not exist", co)
			}
		}

		// Read the results
		for event := range sel.GetStream().Events() {
			var record []byte
			switch event := event.(type) {
			case *types.SelectObjectContentEventStreamMemberRecords:
				record = event.Value.Payload
			}

			if record != nil {
				return fmt.Errorf("unique field %s already exists: %s", uf, record)
			}
		}

		// Close the response body
		sel.GetStream().Close()
	}

	// Create input for GetObject
	getInput := &s3.GetObjectInput{
		Bucket: &c.Bucket,
		Key:    aws.String(co + ".json"),
	}

	// Fetch the object from S3
	get, err := c.Service.GetObject(context.TODO(), getInput)
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return fmt.Errorf("failed to fetch object: %s", err)
		}
	}

	var record []byte

	// Put contents of object into record
	record, err = io.ReadAll(get.Body)
	if err != nil {
		return fmt.Errorf("failed to read object: %s", err)
	}

	// Close the response body
	get.Body.Close()

	obj, err := json.Marshal(i)
	if err != nil {
		return fmt.Errorf("failed to marshal object: %s", err)
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
		return fmt.Errorf("failed to put object: %s", err)
	}

	return nil
}
