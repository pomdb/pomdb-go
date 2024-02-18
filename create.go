package pomdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (c *Client) Create(i interface{}) (*string, error) {
	// Ensure 'i' is a pointer and points to a struct
	rv := reflect.ValueOf(i)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("pomdb: expected pointer to struct, got %T", i)
	}

	// Ensure types of pomdb model fields are correct
	if err := setNewModelFields(i); err != nil {
		return nil, err
	}

	co := getCollectionName(i)

	if ifv := getIndexFieldValues(rv); len(ifv) > 0 {
		if err := c.CheckIndexExists(co, ifv); err != nil {
			return nil, err
		}

		if err := c.CreateIndexItem(co, ifv); err != nil {
			return nil, err
		}
	}

	// Marshal the record
	rec, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	// Get the record's ID
	key := co + "/" + getIdFieldValue(rv)

	put := &s3.PutObjectInput{
		Bucket: &c.Bucket,
		Key:    &key,
		Body:   bytes.NewReader(rec),
	}

	// Set the record's data
	res, err := c.Service.PutObject(context.TODO(), put)
	if err != nil {
		return nil, err
	}

	return res.ETag, nil
}
