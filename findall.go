package pomdb

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type FindAllResult struct {
	Docs      []interface{}
	NextToken string
}

// FindAll returns all objects of a given collection.
func (c *Client) FindAll(q Query) (*FindAllResult, error) {
	// Set default limit
	if q.Limit == 0 {
		q.Limit = QueryLimitDefault
	}

	// Dereference q.Model
	rv, err := dereferenceStruct(q.Model)
	if err != nil {
		return nil, err
	}

	// Build the struct cache
	ca := NewModelCache(rv)

	// Set record prefix path
	pfx := ca.Collection + "/"

	// List all objects concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allObjects []types.Object
	var startAfter *string

	// If the query includes a next token, use it as the starting point
	if q.NextToken != "" {
		startAfter = &q.NextToken
	}

	for {
		lst := &s3.ListObjectsV2Input{
			Bucket:     &c.Bucket,
			Prefix:     &pfx,
			StartAfter: startAfter,
			Delimiter:  aws.String("/"), // Ensures directories are handled correctly
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			pge, err := c.Service.ListObjectsV2(context.TODO(), lst)
			if err != nil {
				return
			}

			mu.Lock()
			for _, obj := range pge.Contents {
				// Filter out directories
				if !strings.HasSuffix(*obj.Key, "/") {
					allObjects = append(allObjects, obj)
				}
			}
			// Set startAfter to the last object key in this batch
			if len(pge.Contents) > 0 {
				startAfter = pge.Contents[len(pge.Contents)-1].Key
			} else {
				startAfter = nil
			}
			mu.Unlock()
		}()

		if startAfter == nil {
			break
		}
	}

	wg.Wait()

	// Filter soft deletes
	if c.SoftDeletes {
		var active []types.Object
		for _, o := range allObjects {
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
				active = append(active, o)
			}
		}

		allObjects = active
	}

	if len(allObjects) == 0 {
		return &FindAllResult{}, nil
	}

	// Apply user-specified or default limit
	var docs []interface{}
	var nextToken string
	for i, obj := range allObjects {
		if i >= q.Limit {
			nextToken = *obj.Key
			break
		}

		get := &s3.GetObjectInput{
			Bucket: &c.Bucket,
			Key:    obj.Key,
		}

		rec, err := c.Service.GetObject(context.TODO(), get)
		if err != nil {
			continue
		}

		elem := reflect.TypeOf(ca.Reference).Elem()
		model := reflect.New(elem).Interface()
		err = json.NewDecoder(rec.Body).Decode(&model)
		if err != nil {
			continue
		}

		docs = append(docs, model)
	}

	return &FindAllResult{
		Docs:      docs,
		NextToken: nextToken,
	}, nil
}
