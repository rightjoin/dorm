package dorm

import (
	"bytes"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/rightjoin/fig"
)

// UploadToS3 uploads the given contents of a file to s3 upon the given key.
// Allowed configurations include:
//    media.s3.region - describes the region for aws
//    media.s3.bucket - describes the bucket for data to be uploaded
//    media.s3.local - true/false; states whether local s3 is to be used
//    media.s3.port - port for the local  running s3
func UploadToS3(raw []byte, key, mime string, size int64) error {
	s3Region := fig.String("media.s3.region")
	s3Bucket := fig.String("media.s3.bucket")

	config := &aws.Config{
		Region: aws.String(s3Region),
	}

	// Support for local S3 (docker image)
	// Prerequisites:
	//   docker run -p 4569:4569 --name xs3 gliffy/fake-s3
	localMode := fig.BoolOr(false, "media.s3.local")
	if localMode {

		// port for the local running docker image, in our case it is 4569
		// defined by passing flag -p
		port := fig.StringOr("4569", "media.s3.port")

		// By default the aws calls are made against the endpoint
		// https://BUCKET.s3.REGION.amazonaws.com/, we need to change
		// it to point against our locally runing docker image
		config.Endpoint = aws.String("http://localhost:" + port)

		// By default "BUCKET.s3.REGION.amazonaws.com" format is used for addressing,
		// but since we are hitting localhost, set the below flag to true,
		// to enforce the use of http://s3.amazonaws.com/BUCKET/KEY format of addressing
		config.S3ForcePathStyle = aws.Bool(true)
	}

	s, err := session.NewSession(config)
	if err != nil {
		return err
	}

	// Prepare upload params
	params := &s3.PutObjectInput{
		Bucket:        aws.String(s3Bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(raw),
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(mime),
	}

	// Upload
	fmt.Printf("Uoloading to s3: %s", key)
	_, err = s3.New(s).PutObject(params)
	return err
}
