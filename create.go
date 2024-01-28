package pomdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Create stores a new object in the S3 bucket. It accepts an input 'i', which
// should be a pointer to a struct. If the struct embeds the Model struct, Create
// initializes the Model's ID field with a new ObjectID, and sets the CreatedAt and
// UpdatedAt fields to the current Unix timestamp. The DeletedAt field is set to zero.
//
// The method performs a uniqueness check if the struct contains a unique field, and
// serializes the struct to JSON before appending it to the corresponding collection
// file in S3. The method decides between a concurrent or regular put operation based
// on the object size.
//
// Create uses reflection to handle the Model struct fields and will return an error
// if 'i' is not a pointer to a struct or if other reflection-based operations fail.
//
// Returns an error if any step in the process fails, including S3 operations, type checks,
// and JSON serialization.
//
// Note: Embedding the Model struct is optional. If not embedded, the method will skip
// setting the Model fields and focus on the S3 storage operations.
//
// Usage:
//
//	type MyStruct struct {
//	    pomdb.Model  // Optional embedding of Model struct
//	    // Other fields...
//	}
//
//	obj := MyStruct{ /* initialize fields */ }
//	err := client.Create(&obj)
//	if err != nil {
//	    // Handle error
//	}
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

		// Execute query
		results, err := c.ExecuteS3SelectQuery(i, query)
		if err != nil {
			return err
		}

		// If results are returned, return error
		if len(results) > 0 {
			return fmt.Errorf("unique field %s already exists", uf)
		}
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
