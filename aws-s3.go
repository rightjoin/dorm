package dorm

import (
	"bytes"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/rightjoin/fig"
)

// UploadToS3 uploads the given contents of a file to s3 across the given key.
func UploadToS3(raw []byte, key string, size int64) error {
	s3Region := fig.StringOr("ap-south-1", "media.s3_region")
	s3Bucket := fig.StringOr("my-bucket", "media.s3_bucket")

	s, err := session.NewSession(&aws.Config{Region: aws.String(s3Region), Endpoint: aws.String("http://localhost:4569"), S3ForcePathStyle: aws.Bool(true)})
	if err != nil {
		return err
	}

	// Upload
	params := &s3.PutObjectInput{
		Bucket:        aws.String(s3Bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(raw),
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(http.DetectContentType(raw)),
	}

	_, err = s3.New(s).PutObject(params)
	return err
}
