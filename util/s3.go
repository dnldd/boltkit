package util

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	S3URLFormat = "https://s3-%s.amazonaws.com/%s/%s"
)

// S3Connection describes a connection to the S3 object storage service.
type S3Connection struct {
	AWSCredentials *credentials.Credentials
	AWSConfig      *aws.Config
	AWSInstance    *s3.S3
	Region         string
}

// NewS3Connection creates a new S3 connection.
func NewS3Connection(accessKey string, secretKey string, region string) (*S3Connection, error) {
	connection := new(S3Connection)
	connection.AWSCredentials = credentials.NewStaticCredentials(accessKey, secretKey, "")
	_, err := connection.AWSCredentials.Get()
	if err != nil {
		return nil, err
	}

	connection.AWSConfig = aws.NewConfig().WithRegion(region).WithCredentials(connection.AWSCredentials)
	connection.AWSInstance = s3.New(session.New(), connection.AWSConfig)
	connection.Region = region
	return connection, nil
}

// CreateS3Bucket creates a bucket if not-existent.
func (connection *S3Connection) CreateBucket(bucketName string) error {
	alreadyCreated := false
	listBucketsParams := &s3.ListBucketsInput{}
	results, err := connection.AWSInstance.ListBuckets(listBucketsParams)
	if err != nil {
		return err
	}

	for _, bucket := range results.Buckets {
		if aws.StringValue(bucket.Name) == bucketName {
			alreadyCreated = true
		}
	}

	if alreadyCreated {
		log.Info("bucket already created, skipping.")
		return nil
	}

	params := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}
	_, err = connection.AWSInstance.CreateBucket(params)
	return err
}

// Upload uploads data to an S3 bucket.
func (connection *S3Connection) Upload(bucketName string, objectPath string, object []byte) (map[string]string, error) {
	fileBytes := bytes.NewReader(object)
	contentType := http.DetectContentType(object)
	contentLength := int64(len(object))
	params := &s3.PutObjectInput{
		ACL:           aws.String("public-read"),
		Bucket:        aws.String(bucketName),
		Key:           aws.String(objectPath),
		Body:          fileBytes,
		ContentLength: aws.Int64(contentLength),
		ContentType:   aws.String(contentType),
	}
	_, err := connection.AWSInstance.PutObject(params)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(S3URLFormat, connection.Region, bucketName, objectPath)
	resourceURL := map[string]string{}
	resourceURL["url"] = url
	return resourceURL, nil
}
