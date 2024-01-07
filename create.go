package pomdb

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

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

	put := &s3.PutObjectInput{
		Bucket: &c.Bucket,
		Key:    aws.String(co + "/" + id.String() + ".json"),
		Body:   strings.NewReader(string(obj)),
	}

	if _, err := c.Service.PutObject(context.TODO(), put); err != nil {
		return err
	}

	return nil
}
