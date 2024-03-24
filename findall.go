package pomdb

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type FindAllResult struct {
	Docs      []interface{}
	NextToken string
}

// FindAll returns all objects of a given collection. It returns an empty Docs
// slice if no objects are found, so it's safe to iterate over the results
// without checking for nil.
func (c *Client) FindAll(q Query) (*FindAllResult, error) {
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
	pfx := ca.Collection + "/"

	lst := &s3.ListObjectsV2Input{
		Bucket:            &c.Bucket,
		Prefix:            &pfx,
		MaxKeys:           &q.Limit,
		Delimiter:         aws.String("/"),
		ContinuationToken: token,
	}

	// Fetch the first page of objects
	page, err := c.Service.ListObjectsV2(context.TODO(), lst)
	if err != nil {
		return nil, err
	}

	// Filter out directories
	var contents []types.Object
	for _, obj := range page.Contents {
		if strings.HasSuffix(*obj.Key, "/") {
			continue
		}

		contents = append(contents, obj)
	}

	// Filter soft deletes
	if c.SoftDeletes {
		var active []types.Object
		for _, obj := range contents {
			tag := &s3.GetObjectTaggingInput{
				Bucket: &c.Bucket,
				Key:    obj.Key,
			}

			tags, err := c.Service.GetObjectTagging(context.TODO(), tag)
			if err != nil {
				return nil, err
			}

			for _, t := range tags.TagSet {
				if *t.Key == "DeletedAt" {
					continue
				}

				active = append(active, obj)
			}

			contents = active
		}
	}

	if len(contents) == 0 {
		return &FindAllResult{}, nil
	}

	// Fetch the list of objects
	var docs []interface{}
	for _, obj := range contents {
		get := &s3.GetObjectInput{
			Bucket: &c.Bucket,
			Key:    obj.Key,
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
	var nextToken string
	if page.NextContinuationToken != nil {
		nextToken = *page.NextContinuationToken
	}

	return &FindAllResult{
		Docs:      docs,
		NextToken: nextToken,
	}, nil
}
