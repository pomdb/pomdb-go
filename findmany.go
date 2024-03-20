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

type FindManyResult struct {
	Contents  []interface{}
	NextToken string
}

func (c *Client) FindMany(q Query) (*FindManyResult, error) {
	if q.Field == "id" {
		return nil, fmt.Errorf("FindMany: cannot search by id")
	}

	// Set default filter
	if q.Filter == "" {
		q.Filter = QueryFilterEquals
	}

	// Set default limit
	if q.Limit == 0 {
		q.Limit = QueryLimitDefault
	}

	// Set the page token
	var token *string
	if q.Token != "" {
		token = &q.Token
	}

	// Dereference q.Model
	rv, err := dereferenceStruct(q.Model)
	if err != nil {
		return nil, err
	}

	// Build the struct cache
	ca := NewModelCache(rv)

	// Set index pfx path
	pfx := ca.Collection + "/indexes/" + q.Field + "/"

	// Fetch the list of objects
	lst := &s3.ListObjectsV2Input{
		Bucket:            &c.Bucket,
		Prefix:            &pfx,
		MaxKeys:           &q.Limit,
		ContinuationToken: token,
	}

	// Fetch the first pge of objects
	pge, err := c.Service.ListObjectsV2(context.TODO(), lst)
	if err != nil {
		return nil, err
	}

	var idxs []map[string]string
	for _, obj := range pge.Contents {
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

	// Find the matches
	err = q.FilterMatches(idxs)
	if err != nil {
		return nil, err
	}

	if len(q.Matches) == 0 {
		log.Println("FindMany: no objects found")
		return nil, nil
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

	// Set the next page token
	var cursor string
	if pge.NextContinuationToken != nil {
		cursor = *pge.NextContinuationToken
	}

	return &FindManyResult{
		Contents:  docs,
		NextToken: cursor,
	}, nil
}
