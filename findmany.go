package pomdb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (c *Client) FindMany(q Query) ([]interface{}, error) {
	if q.FieldName == "id" {
		return nil, fmt.Errorf("FindMany: cannot search by id")
	}

	if q.Filter == "" {
		q.Filter = QueryFilterEquals
	}

	// Dereference q.Model
	rv, err := dereferenceStruct(q.Model)
	if err != nil {
		return nil, err
	}

	// Build the struct cache
	ca := NewModelCache(rv)

	// Get the collection
	co := ca.Collection

	// Set index key path
	key := co + "/indexes/" + q.FieldName + "/"

	// Get a list of all the objects in the index
	lst := &s3.ListObjectsV2Input{
		Bucket: &c.Bucket,
		Prefix: &key,
	}

	// Fetch the list of objects
	var keys []map[string]string
	p := s3.NewListObjectsV2Paginator(c.Service, lst)
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		for _, obj := range page.Contents {
			okey := *obj.Key
			name, err := base64.StdEncoding.DecodeString(okey[len(key):])
			if err != nil {
				return nil, err
			}
			keys = append(keys, map[string]string{
				"key":   okey,
				"value": string(name),
			})
		}
	}

	// Find the matches
	var matches []map[string]string
	for _, k := range keys {
		switch q.Filter {
		case QueryFilterContains:
			if strings.Contains(k["value"], q.FieldValue) {
				matches = append(matches, k)
			}
		case QueryFilterEquals:
			if k["value"] == q.FieldValue {
				matches = append(matches, k)
			}
		case QueryFilterStartsWith:
			if strings.HasPrefix(k["value"], q.FieldValue) {
				matches = append(matches, k)
			}
		case QueryFilterEndsWith:
			if strings.HasSuffix(k["value"], q.FieldValue) {
				matches = append(matches, k)
			}
		case QueryFilterGreaterThan:
			if k["value"] > q.FieldValue {
				matches = append(matches, k)
			}
		case QueryFilterLessThan:
			if k["value"] < q.FieldValue {
				matches = append(matches, k)
			}
		default:
			return nil, fmt.Errorf("FindMany: invalid filter")
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("FindMany: no matches found")
	}

	// Fetch the records
	var objs []interface{}
	for _, m := range matches {
		get := &s3.GetObjectInput{
			Bucket: &c.Bucket,
			Key:    aws.String(m["key"]),
		}

		rec, err := c.Service.GetObject(context.TODO(), get)
		if err != nil {
			return nil, err
		}

		bdy, err := io.ReadAll(rec.Body)
		if err != nil {
			return nil, err
		}

		id := string(bdy)

		key = co + "/" + id

		get = &s3.GetObjectInput{
			Bucket: &c.Bucket,
			Key:    &key,
		}

		rec, err = c.Service.GetObject(context.TODO(), get)
		if err != nil {
			return nil, err
		}

		if err = json.NewDecoder(rec.Body).Decode(&q.Model); err != nil {
			return nil, err
		}

		objs = append(objs, q.Model)
	}

	return objs, nil
}
