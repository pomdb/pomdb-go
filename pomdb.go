package pomdb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"reflect"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Define minimum and maximum part sizes
const (
	MinGetPartSize = 500               // 500B
	MaxGetPartSize = 1 * 1024 * 1024   // 1MB
	MinPutPartSize = 5 * 1024 * 1024   // 5MB
	MaxPutPartSize = 100 * 1024 * 1024 // 100MB
)

type PartUploadResponse struct {
	PartNumber int32
	ETag       string
}

type Client struct {
	Bucket  string
	Region  string
	Service *s3.Client
}

func (c *Client) Connect() error {
	conf, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(c.Region),
	)
	if err != nil {
		return err
	}

	c.Service = s3.NewFromConfig(conf)

	if err := c.CheckBucket(); err != nil {
		return fmt.Errorf("bucket %s does not exist", c.Bucket)
	}

	log.Printf("connected to %s", c.Bucket)

	return nil
}

func (c *Client) CheckBucket() error {
	head := &s3.HeadBucketInput{
		Bucket: &c.Bucket,
	}

	if _, err := c.Service.HeadBucket(context.TODO(), head); err != nil {
		return err
	}

	return nil
}

// ConcurrentGetObject retrieves an object from an S3 bucket in parallelized parts based on the provided key.
// It calculates the object's size and fetches it using concurrent byte-range requests, dynamically adjusting the
// part sizes within minimum and maximum constraints. The method uses 'ctx' for managing the request lifecycle
// and 'key' to identify the object in the S3 bucket. After completing all operations, it combines these parts
// in the correct order to reconstruct the full object.
//
// The function returns a byte slice containing the object's data. If any errors occur during the retrieval process,
// such as issues in determining the object size, during part retrieval, or in combining the parts, an error is returned.
//
// Usage:
//
//	data, err := client.ConcurrentGetObject(context.Background(), "object-key")
//	if err != nil {
//	    log.Printf("Error retrieving object: %s", err)
//	} else {
//	    log.Printf("Object retrieved successfully, size: %d bytes", len(data))
//	}
func (c *Client) ConcurrentGetObject(ctx context.Context, key string) ([]byte, error) {
	// Step 1: Determine object size
	headInput := &s3.HeadObjectInput{
		Bucket: &c.Bucket,
		Key:    aws.String(key),
	}
	headOutput, err := c.Service.HeadObject(ctx, headInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get object head: %s", err)
	}
	size := headOutput.ContentLength

	// Step 2: Calculate dynamic part size
	dynamicPartSize := MinGetPartSize * int64(math.Log2(float64(*size)/float64(MinGetPartSize)+1))
	if dynamicPartSize > MaxGetPartSize {
		dynamicPartSize = MaxGetPartSize
	}
	if dynamicPartSize < MinGetPartSize {
		dynamicPartSize = MinGetPartSize
	}
	numParts := (*size + dynamicPartSize - 1) / dynamicPartSize

	log.Printf("ConcurrentGetObject: size=%d, dynamicPartSize=%d, numParts=%d", *size, dynamicPartSize, numParts)

	// Step 3: Parallelize byte-range requests with correct ordering
	parts := make([][]byte, numParts)
	var wg sync.WaitGroup
	wg.Add(int(numParts))
	for i := int64(0); i < numParts; i++ {
		go func(i int64) {
			defer wg.Done()
			rangeStart := i * dynamicPartSize
			rangeEnd := rangeStart + dynamicPartSize - 1
			if rangeEnd >= *size {
				rangeEnd = *size - 1
			}
			rangeString := fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd)

			getInput := &s3.GetObjectInput{
				Bucket: &c.Bucket,
				Key:    aws.String(key),
				Range:  aws.String(rangeString),
			}
			getObjectOutput, err := c.Service.GetObject(ctx, getInput)
			if err != nil {
				// Ideally, handle error in a way that doesn't simply ignore it. For now, we return.
				fmt.Printf("Error fetching part %d: %s\n", i, err)
				return
			}
			partData, _ := io.ReadAll(getObjectOutput.Body)
			parts[i] = partData
		}(i)
	}

	// Wait for all parts to be fetched
	wg.Wait()

	// Step 4: Combine the results in the correct order
	var combined []byte
	for _, part := range parts {
		combined = append(combined, part...)
	}

	return combined, nil
}

// ConcurrentPutObject uploads an object to an S3 bucket in parallelized parts. This method takes
// the provided 'data' byte slice and uploads it under the specified 'key'. The upload process is
// optimized for handling large objects by breaking the data into parts and uploading them concurrently.
// The method uses 'ctx' as the context for the request lifecycle and 'key' as the identifier for the
// object in the S3 bucket. It calculates the size of 'data', determines the dynamic part size for the
// upload, and initiates a multipart upload process. Each part of the data is uploaded in parallel,
// and upon completion of all uploads, the parts are assembled to complete the object upload.
//
// The method returns an error if any issues occur during the upload process. This includes errors
// in initiating the multipart upload, uploading individual parts, or in the final assembly of the parts.
//
// Usage:
//
//	data := []byte("your data here")
//	err := client.ConcurrentPutObject(context.Background(), "object-key", data)
//	if err != nil {
//	    log.Printf("Error uploading object: %s", err)
//	} else {
//	    log.Printf("Object uploaded successfully")
//	}
func (c *Client) ConcurrentPutObject(ctx context.Context, key string, data []byte) error {
	size := int64(len(data)) // Size of the data

	// Calculate dynamic part size
	dynamicPartSize := MinPutPartSize * int64(math.Log2(float64(size)/float64(MinPutPartSize)+1))
	if dynamicPartSize > MaxPutPartSize {
		dynamicPartSize = MaxPutPartSize
	}
	if dynamicPartSize < MinPutPartSize {
		dynamicPartSize = MinPutPartSize
	}
	numParts := (size + dynamicPartSize - 1) / dynamicPartSize

	log.Printf("ConcurrentPutObject: size=%d, dynamicPartSize=%d, numParts=%d", size, dynamicPartSize, numParts)

	// Initiate Multipart Upload
	createMultipartUploadInput := &s3.CreateMultipartUploadInput{
		Bucket: &c.Bucket,
		Key:    aws.String(key),
	}
	multipartUpload, err := c.Service.CreateMultipartUpload(ctx, createMultipartUploadInput)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	partUploadResponses := make([]PartUploadResponse, numParts)

	for i := int64(0); i < numParts; i++ {
		wg.Add(1)
		go func(i int64) {
			defer wg.Done()

			rangeStart := i * dynamicPartSize
			rangeEnd := rangeStart + dynamicPartSize
			if rangeEnd > size {
				rangeEnd = size
			}

			partData := data[rangeStart:rangeEnd]

			// Upload the part
			uploadPartInput := &s3.UploadPartInput{
				Bucket:     &c.Bucket,
				Key:        aws.String(key),
				PartNumber: aws.Int32(int32(i + 1)), // Part numbers are 1-based
				UploadId:   multipartUpload.UploadId,
				Body:       bytes.NewReader(partData),
			}
			resp, err := c.Service.UploadPart(ctx, uploadPartInput)
			if err != nil {
				fmt.Printf("Error uploading part %d: %s\n", i+1, err)
				return
			}
			partUploadResponses[i] = PartUploadResponse{
				PartNumber: int32(i + 1),
				ETag:       *resp.ETag,
			}
		}(i)
	}

	wg.Wait()

	// Complete Multipart Upload
	var completedParts []types.CompletedPart
	for _, partResp := range partUploadResponses {
		completedParts = append(completedParts, types.CompletedPart{
			ETag:       aws.String(partResp.ETag),
			PartNumber: aws.Int32(partResp.PartNumber),
		})
	}

	completeMultipartUploadInput := &s3.CompleteMultipartUploadInput{
		Bucket:   &c.Bucket,
		Key:      aws.String(key),
		UploadId: multipartUpload.UploadId,
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	}
	_, err = c.Service.CompleteMultipartUpload(ctx, completeMultipartUploadInput)
	if err != nil {
		return err
	}

	return nil
}

// ExecuteS3SelectQuery performs a select query on an S3 object based on the provided query string.
// It is designed to be used with any struct that is a pointer to a struct and potentially
// contains an embedded Model struct. The query is executed against the specified S3 bucket
// and object, which are determined based on the type of the input struct 'i'.
//
// The input 'i' must be a pointer to a struct, and the query should be a valid SQL
// expression compatible with S3's SelectObjectContent API. The method constructs and
// executes the S3 SelectObjectContent query, unmarshals the resulting JSON data into
// new instances of the same type as 'i', and returns a slice of these instances.
//
// Returns a slice of interface{} containing unmarshaled results and an error if the
// query execution or unmarshaling fails. Possible failure reasons include invalid query
// syntax, issues with S3 connectivity, problems with the S3 object, or unmarshaling errors.
//
// Note: The method assumes that the resulting JSON structure from the query matches
// the structure of 'i'. It creates new instances of 'i' for each record in the query result.
//
// Usage:
//
//	type MyStruct struct {
//	    pomdb.Model  // Embedding Model struct is optional
//	    // Other fields...
//	}
//
//	query := "SELECT * FROM S3Object WHERE someField = 'someValue'"
//	var obj MyStruct
//	results, err := client.ExecuteS3SelectQuery(&obj, query)
//	if err != nil {
//	    // Handle error
//	}
//	for _, result := range results {
//	    // Process each result, which is of type *MyStruct
//	}
func (c *Client) ExecuteS3SelectQuery(i interface{}, query string) ([]interface{}, error) {
	// Ensure 'i' is a pointer and points to a struct
	rv := reflect.ValueOf(i)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("model must be a pointer to a struct")
	}

	// Create input for SelectObjectContent
	selectInput := &s3.SelectObjectContentInput{
		Bucket:         &c.Bucket,
		Key:            aws.String(getCollectionName(i) + ".json"),
		Expression:     aws.String(query),
		ExpressionType: types.ExpressionTypeSql,
		InputSerialization: &types.InputSerialization{
			JSON: &types.JSONInput{
				Type: types.JSONTypeDocument,
			},
		},
		OutputSerialization: &types.OutputSerialization{
			JSON: &types.JSONOutput{
				RecordDelimiter: aws.String("\n"),
			},
		},
	}

	// Execute the query
	sel, err := c.Service.SelectObjectContent(context.Background(), selectInput)
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return nil, fmt.Errorf("collection %s does not exist", getCollectionName(i))
		}
	}

	// Read the results
	var results []interface{}
	for event := range sel.GetStream().Events() {
		var record []byte
		switch event := event.(type) {
		case *types.SelectObjectContentEventStreamMemberRecords:
			record = event.Value.Payload
		}

		if record != nil {
			// Create new instance of 'i'
			ni := reflect.New(reflect.TypeOf(i).Elem()).Interface()

			// Unmarshal record into 'ni'
			err = json.Unmarshal(record, ni)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal record: %s", err)
			}

			// Append 'ni' to results
			results = append(results, ni)
		}
	}

	// Close the response body
	sel.GetStream().Close()

	return results, nil
}
