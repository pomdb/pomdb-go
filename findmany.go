package pomdb

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

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

	// Set default filter
	if q.Filter == nil {
		def := QueryFilterDefault
		q.Filter = &def
	}

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

	// Get the index field
	var idx *IndexField
	for _, i := range ca.IndexFields {
		if i.FieldName == q.Field {
			idx = &i
			break
		}
	}
	if idx == nil {
		return nil, fmt.Errorf("FindMany: index field %s not found", q.Field)
	}

	// Set index prefix path
	pfx, err := encodeQueryPrefix(ca.Collection, q.Field, idx.IndexType)
	if err != nil {
		return nil, err
	}

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
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			pge, err := c.Service.ListObjectsV2(context.TODO(), lst)
			if err != nil {
				return
			}

			mu.Lock()
			allObjects = append(allObjects, pge.Contents...)
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

	// Filter soft-deletes
	if c.SoftDeletes {
		var contents []types.Object
		for _, o := range allObjects {
			tag := &s3.GetObjectTaggingInput{
				Bucket: &c.Bucket,
				Key:    o.Key,
			}

			tags, err := c.Service.GetObjectTagging(context.TODO(), tag)
			if err != nil {
				continue
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

		allObjects = contents
	}

	// Apply query filters
	if q.Filter != nil {
		var contents []types.Object
		for _, obj := range allObjects {
			res, err := q.Compare(obj, idx)
			if err != nil {
				return nil, err
			}
			if res {
				contents = append(contents, obj)
			}
		}
		allObjects = contents
	}

	// Apply user-specified or default limit
	var docs []interface{}
	var nextToken string
	for i, o := range allObjects {
		if i >= q.Limit {
			nextToken = *o.Key
			break
		}

		uid := strings.Split(*o.Key, "/")[5]

		get := &s3.GetObjectInput{
			Bucket: &c.Bucket,
			Key:    aws.String(ca.Collection + "/" + uid),
		}

		doc, err := c.Service.GetObject(context.TODO(), get)
		if err != nil {
			continue
		}

		elem := reflect.TypeOf(ca.Reference).Elem()
		model := reflect.New(elem).Interface()
		err = json.NewDecoder(doc.Body).Decode(&model)
		if err != nil {
			continue
		}

		docs = append(docs, model)
	}

	return &FindManyResult{
		Docs:      docs,
		NextToken: nextToken,
	}, nil
}
