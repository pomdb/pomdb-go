package pomdb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"reflect"

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

	// Set index pfx path
	pfx := ca.Collection + "/indexes/" + q.FieldName + "/"

	// Fetch the list of objects
	lst := &s3.ListObjectsV2Input{
		Bucket: &c.Bucket,
		Prefix: &pfx,
	}

	// Fetch the list of objects
	var idxs []map[string]string
	pgr := s3.NewListObjectsV2Paginator(c.Service, lst)
	for pgr.HasMorePages() {
		page, err := pgr.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		for _, obj := range page.Contents {
			key := *obj.Key
			name, err := base64.StdEncoding.DecodeString(key[len(pfx):])
			if err != nil {
				return nil, err
			}
			idxs = append(idxs, map[string]string{
				"key":   key,
				"value": string(name),
			})
		}
	}

	// Find the matches
	err = q.FilterMatches(idxs)
	if err != nil {
		return nil, err
	}

	if len(q.Matches) == 0 {
		log.Println("no matches")
		return []interface{}{}, nil
	}

	// Fetch the documents
	var docs []interface{}
	for _, m := range q.Matches {
		get := &s3.GetObjectInput{
			Bucket: &c.Bucket,
			Key:    aws.String(m["key"]),
		}

		idx, err := c.Service.GetObject(context.TODO(), get)
		if err != nil {
			return nil, err
		}

		bdy, err := io.ReadAll(idx.Body)
		if err != nil {
			return nil, err
		}

		get = &s3.GetObjectInput{
			Bucket: &c.Bucket,
			Key:    aws.String(ca.Collection + "/" + string(bdy)),
		}

		rec, err := c.Service.GetObject(context.TODO(), get)
		if err != nil {
			return nil, err
		}

		elem := reflect.TypeOf(q.Model).Elem()
		model := reflect.New(elem).Interface()
		err = json.NewDecoder(rec.Body).Decode(&model)
		if err != nil {
			return nil, err
		}

		docs = append(docs, model)
	}

	return docs, nil
}
