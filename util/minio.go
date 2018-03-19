package util

import (
	"bytes"
	"fmt"

	minio "github.com/minio/minio-go"
)

const (
	DOSPURLFormat = "https://%s.%s.digitaloceanspaces.com/%s"
	DOSPLocation  = "us-east-1"
)

type Minio struct {
	Client *minio.Client
	Region string
	Bucket string
}

// NewMinio creates a new minio client.
func NewMinio(region string, endpoint string, accessKey string, secretKey string) (*Minio, error) {
	var err error
	mc := new(Minio)
	mc.Region = region
	url := fmt.Sprintf("%s.%s", region, endpoint)
	mc.Client, err = minio.NewWithRegion(url, accessKey, secretKey, true, region)
	if err != nil {
		return nil, err
	}

	return mc, nil
}

// CreateBucket creates the the provided bucket if non-existent.
func (mc *Minio) CreateBucket(bucketName string) error {
	mc.Bucket = bucketName
	exists, _ := mc.Client.BucketExists(bucketName)
	if exists {
		log.Info("bucket already created, skipping.")
		return nil
	}

	err := mc.Client.MakeBucket(bucketName, DOSPLocation)
	if err != nil {
		return err
	}

	return nil
}

// Upload uploads data to the specified bucket.
func (mc *Minio) Upload(objectPath string, object *[]byte) (*map[string]string, error) {
	mime := GetMime(object)
	_, err := mc.Client.PutObject(mc.Bucket, objectPath, bytes.NewReader(*object), int64(len(*object)), minio.PutObjectOptions{ContentType: mime})
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(DOSPURLFormat, mc.Bucket, mc.Region, objectPath)
	resourceURL := map[string]string{}
	resourceURL["url"] = url
	return &resourceURL, nil
}
