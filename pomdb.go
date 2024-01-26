package pomdb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math"
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
