package blob

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

type S3Options struct{ Endpoint, Region, Bucket, AccessKey, SecretKey string }
type S3 struct {
	client *s3.Client
	bucket string
}

const objectPrefix = "suining-images/"

func ValidateS3(o S3Options) error {
	u, err := url.Parse(o.Endpoint)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return errors.New("invalid S3 endpoint")
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme != "https" && !(u.Scheme == "http" && ip != nil && ip.IsLoopback()) {
		return errors.New("S3 endpoint requires HTTPS except numeric loopback test servers")
	}
	if o.Region == "" || o.Bucket == "" || o.AccessKey == "" || o.SecretKey == "" {
		return errors.New("S3 region, private bucket and credentials are required")
	}
	return nil
}
func NewS3(o S3Options) (*S3, error) {
	if err := ValidateS3(o); err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(aws.Config{Region: o.Region, Credentials: credentials.NewStaticCredentialsProvider(o.AccessKey, o.SecretKey, ""), HTTPClient: &http.Client{Timeout: 30 * time.Second}}, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(strings.TrimSuffix(o.Endpoint, "/"))
		options.UsePathStyle = true
		options.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		options.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})
	return &S3{client: client, bucket: o.Bucket}, nil
}
func (s *S3) Put(ctx context.Context, key string, data []byte) error {
	if !ValidKey(key) {
		return ErrNotFound
	}
	if len(data) > MaxStoredBytes {
		return ErrIntegrity
	}
	sum := sha256.Sum256(data)
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(objectPrefix + key), Body: bytes.NewReader(data), ContentLength: aws.Int64(int64(len(data))), ContentType: aws.String("application/octet-stream"), IfNoneMatch: aws.String("*"), ChecksumSHA256: aws.String(base64.StdEncoding.EncodeToString(sum[:]))})
	var api smithy.APIError
	if errors.As(err, &api) && api.ErrorCode() == "PreconditionFailed" {
		return ErrExists
	}
	return err
}
func (s *S3) Get(ctx context.Context, key string) ([]byte, error) {
	if !ValidKey(key) {
		return nil, ErrNotFound
	}
	object, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(objectPrefix + key)})
	var api smithy.APIError
	if errors.As(err, &api) && (api.ErrorCode() == "NoSuchKey" || api.ErrorCode() == "NotFound") {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	defer object.Body.Close()
	if object.ContentLength != nil && *object.ContentLength > MaxStoredBytes {
		return nil, ErrIntegrity
	}
	data, err := io.ReadAll(io.LimitReader(object.Body, MaxStoredBytes+1))
	if len(data) > MaxStoredBytes {
		return nil, ErrIntegrity
	}
	return data, err
}
func (s *S3) Delete(ctx context.Context, key string) error {
	if !ValidKey(key) {
		return ErrNotFound
	}
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(objectPrefix + key)})
	return err
}
func (s *S3) Walk(ctx context.Context, visit func(Entry) error) error {
	pages := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{Bucket: aws.String(s.bucket), Prefix: aws.String(objectPrefix), MaxKeys: aws.Int32(256)})
	for pages.HasMorePages() {
		page, err := pages.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, item := range page.Contents {
			key := strings.TrimPrefix(aws.ToString(item.Key), objectPrefix)
			if !ValidKey(key) {
				continue
			}
			if err := visit(Entry{key, aws.ToInt64(item.Size), aws.ToTime(item.LastModified)}); err != nil {
				return err
			}
		}
	}
	return nil
}
