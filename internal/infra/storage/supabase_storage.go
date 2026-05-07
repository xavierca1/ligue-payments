package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SupabaseStorage uploads files to a Supabase Storage bucket via the REST API.
// Uses the service role key, so uploads bypass RLS — keep the key server-side only.
type SupabaseStorage struct {
	projectURL string // e.g. "https://yntprscrhdlrwkgnmzrb.supabase.co"
	bucket     string // e.g. "contracts"
	serviceKey string // Supabase service_role JWT
	client     *http.Client
}

func NewSupabaseStorage(projectURL, bucket, serviceKey string) *SupabaseStorage {
	return &SupabaseStorage{
		projectURL: projectURL,
		bucket:     bucket,
		serviceKey: serviceKey,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

// Upload sends data to Supabase Storage at the given object path.
// x-upsert: true overwrites any existing file at that path.
// Returns the public URL of the uploaded file.
func (s *SupabaseStorage) Upload(ctx context.Context, path string, data []byte) (string, error) {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.projectURL, s.bucket, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("build upload request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("Content-Type", "application/pdf")
	req.Header.Set("x-upsert", "true")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("supabase upload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("supabase upload failed (HTTP %d): %s", resp.StatusCode, body)
	}

	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.projectURL, s.bucket, path)
	return publicURL, nil
}
