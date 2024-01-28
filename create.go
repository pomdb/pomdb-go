package pomdb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func (c *Client) Create(i interface{}) error {
	// Ensure 'i' is a pointer and points to a struct
	rv := reflect.ValueOf(i)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("model must be a pointer to a struct")
	}

	// Set model fields
	if field := rv.FieldByName("ID"); field.IsValid() && field.CanSet() {
		field.Set(reflect.ValueOf(NewObjectID()))
	}
	now := time.Now().Unix()
	if field := rv.FieldByName("CreatedAt"); field.IsValid() && field.CanSet() {
		field.SetInt(now)
	}
	if field := rv.FieldByName("UpdatedAt"); field.IsValid() && field.CanSet() {
		field.SetInt(now)
	}
	if field := rv.FieldByName("DeletedAt"); field.IsValid() && field.CanSet() {
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

	record, err := c.ConcurrentGetObject(context.TODO(), co+".json")
	if err != nil {
		return fmt.Errorf("failed to fetch object: %s", err)
	}

	obj, err := json.Marshal(i)
	if err != nil {
		return fmt.Errorf("failed to marshal object: %s", err)
	}

	// Append newline to object
	obj = append(obj, []byte("\n")...)

	// Append object to record
	record = append(record, obj...)

	// If > MinPutPartSize, use concurrent put
	if len(record) > MinPutPartSize {
		err = c.ConcurrentPutObject(context.TODO(), co+".json", record)
		if err != nil {
			return fmt.Errorf("failed to put object: %s", err)
		}

		return nil
	}

	// Otherwise, use regular put object
	putInput := &s3.PutObjectInput{
		Bucket: &c.Bucket,
		Key:    aws.String(co + ".json"),
		Body:   bytes.NewReader(record),
	}

	_, err = c.Service.PutObject(context.TODO(), putInput)
	if err != nil {
		return fmt.Errorf("failed to put object: %s", err)
	}

	return nil
}
