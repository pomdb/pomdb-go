package pomdb

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type FindManyResult struct {
	Docs      []interface{}
	NextToken string
}

// FindMany retrieves multiple objects of a given index.
func (c *Client) FindMany(q Query) (*FindManyResult, error) {
	if q.Field == "id" {
		return nil, fmt.Errorf("FindMany: cannot search by id")
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
	pfx, err := encodeIndexPrefix(ca.Collection, q.Field, q.Value, false)
	if err != nil {
		return nil, err
	}

	lst := &s3.ListObjectsV2Input{
		Bucket:  &c.Bucket,
		Prefix:  &pfx,
		MaxKeys: &q.Limit,
	}

	if token != nil {
		lst.ContinuationToken = token
	}

	pge, err := c.Service.ListObjectsV2(context.TODO(), lst)
	if err != nil {
		return nil, err
	}

	// Filter soft-deletes
	if c.SoftDeletes {
		var contents []types.Object
		for _, o := range pge.Contents {
			tag := &s3.GetObjectTaggingInput{
				Bucket: &c.Bucket,
				Key:    o.Key,
			}

			tags, err := c.Service.GetObjectTagging(context.TODO(), tag)
			if err != nil {
				return nil, err
			}

			deleted := false
			for _, t := range tags.TagSet {
				if *t.Key == "DeletedAt" {
					deleted = true
					break
				}
			}

			if !deleted {
				contents = append(contents, o)
			}
		}

		pge.Contents = contents
	}

	// Check for no results
	if len(pge.Contents) == 0 {
		return &FindManyResult{}, nil
	}

	// Fetch the documents
	var docs []interface{}
	for _, o := range pge.Contents {
		uid := strings.TrimPrefix(*o.Key, pfx+"/")

		get := &s3.GetObjectInput{
			Bucket: &c.Bucket,
			Key:    aws.String(ca.Collection + "/" + uid),
		}

		doc, err := c.Service.GetObject(context.TODO(), get)
		if err != nil {
			return nil, err
		}

		elem := reflect.TypeOf(q.Model).Elem()
		model := reflect.New(elem).Interface()
		err = json.NewDecoder(doc.Body).Decode(&model)
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
		Docs:      docs,
		NextToken: cursor,
	}, nil
}
