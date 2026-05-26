package objectstore

import (
	"context"
	"time"

	"github.com/itshop/api/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewClient(cfg config.Config) (*minio.Client, error) {
	return minio.New(cfg.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure: cfg.S3UseSSL,
		Region: "us-east-1",
	})
}

func EnsureBucket(ctx context.Context, c *minio.Client, name string) error {
	ok, err := c.BucketExists(ctx, name)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return c.MakeBucket(ctx, name, minio.MakeBucketOptions{Region: "us-east-1"})
}

func PresignedGET(ctx context.Context, c *minio.Client, bucket, key string, ttl time.Duration) (string, error) {
	u, err := c.PresignedGetObject(ctx, bucket, key, ttl, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
