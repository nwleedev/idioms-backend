package lib

import (
	"io"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Storage struct {
	S3     func() *s3.Client
	Bucket string
}

type File struct {
	Content   io.Reader
	Extension string
}
