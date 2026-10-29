package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/tafakorpardazan/huginn/internal/config"
)

// Object describes a stored file.
type Object struct {
	Key              string    `json:"object_key"`
	Bucket           string    `json:"bucket"`
	MimeType         string    `json:"mime_type"`
	OriginalFilename string    `json:"original_filename"`
	Size             int64     `json:"size"`
	Checksum         string    `json:"checksum"`
	UploadedBy       string    `json:"uploaded_by"`
	CreatedAt        time.Time `json:"created_at"`
}

// Store is the MinIO-backed file storage.
type Store struct {
	client *minio.Client
	bucket string
	ttl    time.Duration
}

func New(ctx context.Context, cfg config.MinIOConfig) (*Store, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("minio bucket check: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("minio create bucket: %w", err)
		}
	}
	return &Store{
		client: client,
		bucket: cfg.Bucket,
		ttl:    time.Duration(cfg.PresignTTLHours) * time.Hour,
	}, nil
}

// Put stores a reader stream under a key in the configured bucket.
func (s *Store) Put(ctx context.Context, r io.Reader, size int64, contentType, objectKey string) (*Object, error) {
	hasher := sha256.New()
	tee := io.TeeReader(r, hasher)

	info, err := s.client.PutObject(ctx, s.bucket, objectKey, tee, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, fmt.Errorf("minio put object: %w", err)
	}

	return &Object{
		Key:       objectKey,
		Bucket:    s.bucket,
		MimeType:  contentType,
		Size:      info.Size,
		Checksum:  hex.EncodeToString(hasher.Sum(nil)),
		CreatedAt: time.Now().UTC(),
	}, nil
}

// Get streams an object by key.
func (s *Store) Get(ctx context.Context, objectKey string) (io.ReadCloser, int64, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("minio get object: %w", err)
	}
	stat, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, 0, fmt.Errorf("minio stat object: %w", err)
	}
	return obj, stat.Size, nil
}

// Stat returns object metadata without downloading.
func (s *Store) Stat(ctx context.Context, objectKey string) (minio.ObjectInfo, error) {
	return s.client.StatObject(ctx, s.bucket, objectKey, minio.StatObjectOptions{})
}

// PresignedGetURL returns a temporary download URL for a key.
func (s *Store) PresignedGetURL(ctx context.Context, objectKey string) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucket, objectKey, s.ttl, nil)
	if err != nil {
		return "", fmt.Errorf("minio presign: %w", err)
	}
	return u.String(), nil
}

func (s *Store) Delete(ctx context.Context, objectKey string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
}

func (s *Store) Bucket() string { return s.bucket }

// NewObjectKey builds a namespaced object key. The userID is the uploader id.
func NewObjectKey(prefix, originalFilename string) string {
	ext := strings.ToLower(path.Ext(originalFilename))
	name := strings.TrimSuffix(path.Base(originalFilename), ext)
	ext = strings.ToLower(ext)
	return fmt.Sprintf("%s/%s_%s%s", prefix, uuid.NewString(), safe(name), ext)
}

func safe(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "file"
	}
	return b.String()
}
