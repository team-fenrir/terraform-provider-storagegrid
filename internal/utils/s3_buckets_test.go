// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestDefaultRetentionSettingUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    DefaultRetentionSetting
		wantErr bool
	}{
		{
			name:  "numeric days",
			input: `{"mode":"governance","days":30}`,
			want:  DefaultRetentionSetting{Mode: "governance", Days: 30},
		},
		{
			name:  "string days",
			input: `{"mode":"governance","days":"45"}`,
			want:  DefaultRetentionSetting{Mode: "governance", Days: 45},
		},
		{
			name:  "numeric years",
			input: `{"mode":"compliance","years":2}`,
			want:  DefaultRetentionSetting{Mode: "compliance", Years: 2},
		},
		{
			name:  "empty strings are ignored",
			input: `{"mode":"governance","days":"","years":""}`,
			want:  DefaultRetentionSetting{Mode: "governance"},
		},
		{
			name:    "invalid json",
			input:   `{`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got DefaultRetentionSetting
			err := json.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestDefaultRetentionSettingMarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		value DefaultRetentionSetting
		want  map[string]any
	}{
		{
			name:  "days",
			value: DefaultRetentionSetting{Mode: "governance", Days: 30},
			want:  map[string]any{"mode": "governance", "days": float64(30)},
		},
		{
			name:  "zero days is still sent when years is unset",
			value: DefaultRetentionSetting{Mode: "governance"},
			want:  map[string]any{"mode": "governance", "days": float64(0)},
		},
		{
			name:  "years takes priority",
			value: DefaultRetentionSetting{Mode: "compliance", Days: 30, Years: 2},
			want:  map[string]any{"mode": "compliance", "years": float64(2)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes, err := json.Marshal(&tt.value)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var got map[string]any
			if err := json.Unmarshal(bytes, &got); err != nil {
				t.Fatalf("failed to unmarshal output %q: %v", string(bytes), err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestGetCachedBucketListUsesIncludeQueryAndCache(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/api/v4/org/containers" {
			t.Fatalf("path = %s, want /api/v4/org/containers", r.URL.Path)
		}
		if got := r.URL.Query().Get("include"); got != bucketListIncludeParams {
			t.Fatalf("include query = %q, want %q", got, bucketListIncludeParams)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want Bearer test-token", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"apiVersion": "4.0",
			"data": [
				{"name": "logs", "creationTime": "2026-01-01T00:00:00.000Z", "region": "us-east-1"}
			]
		}`))
	}))
	defer server.Close()

	client := &Client{
		EndpointURL: server.URL,
		HTTPClient:  server.Client(),
		Token:       "test-token",
	}

	for i := range 2 {
		bucket, err := client.GetS3Bucket("logs")
		if err != nil {
			t.Fatalf("GetS3Bucket attempt %d returned error: %v", i+1, err)
		}
		if bucket.Name != "logs" || bucket.Region != "us-east-1" {
			t.Fatalf("bucket = %#v", bucket)
		}
	}
	if requests != 1 {
		t.Fatalf("server received %d requests, want 1 cached request", requests)
	}
}

func TestGetCachedBucketListRefreshesExpiredCache(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":[{"name":"logs"}]}`))
	}))
	defer server.Close()

	client := &Client{
		EndpointURL:     server.URL,
		HTTPClient:      server.Client(),
		Token:           "test-token",
		bucketCache:     []S3BucketData{{Name: "logs"}},
		bucketCacheTime: time.Now().Add(-6 * time.Minute),
	}

	if _, err := client.GetS3Bucket("logs"); err != nil {
		t.Fatalf("GetS3Bucket returned error: %v", err)
	}
	if requests != 1 {
		t.Fatalf("server received %d requests, want 1 refresh request", requests)
	}
}

func TestCreateS3BucketRequest(t *testing.T) {
	tests := []struct {
		name              string
		objectLockEnabled bool
		wantObjectLock    S3BucketCreateObjectLock
	}{
		{
			name:              "without object lock",
			objectLockEnabled: false,
			wantObjectLock:    S3BucketCreateObjectLock{Enabled: false},
		},
		{
			name:              "with object lock",
			objectLockEnabled: true,
			wantObjectLock: S3BucketCreateObjectLock{
				Enabled: true,
				DefaultRetentionSetting: &S3BucketCreateRetentionSetting{
					Mode: "governance",
					Days: 1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
				}
				if r.URL.Path != "/api/v4/org/containers" {
					t.Fatalf("path = %s, want /api/v4/org/containers", r.URL.Path)
				}
				if got := r.Header.Get("Content-Type"); got != "application/json" {
					t.Fatalf("Content-Type = %q, want application/json", got)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
					t.Fatalf("Authorization = %q, want Bearer test-token", got)
				}

				var got S3BucketCreateRequest
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}
				if got.Name != "logs" || got.Region != "us-east-1" {
					t.Fatalf("request = %#v", got)
				}
				if got.S3ObjectLock == nil {
					t.Fatal("S3ObjectLock is nil")
				}
				if !reflect.DeepEqual(*got.S3ObjectLock, tt.wantObjectLock) {
					t.Fatalf("S3ObjectLock = %#v, want %#v", *got.S3ObjectLock, tt.wantObjectLock)
				}

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"success","data":{"name":"logs","region":"us-east-1"}}`))
			}))
			defer server.Close()

			client := &Client{
				EndpointURL:     server.URL,
				HTTPClient:      server.Client(),
				Token:           "test-token",
				bucketCache:     []S3BucketData{{Name: "stale"}},
				bucketCacheTime: time.Now(),
				S3EndpointURL:   "https://s3.example.com",
			}

			if err := client.CreateS3Bucket("logs", "us-east-1", tt.objectLockEnabled); err != nil {
				t.Fatalf("CreateS3Bucket returned error: %v", err)
			}
			if client.bucketCache != nil {
				t.Fatalf("bucketCache = %#v, want nil after create", client.bucketCache)
			}
			if !client.bucketCacheTime.IsZero() {
				t.Fatalf("bucketCacheTime = %v, want zero after create", client.bucketCacheTime)
			}
		})
	}
}

func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil",
			err:  nil,
			want: false,
		},
		{
			name: "context deadline",
			err:  context.DeadlineExceeded,
			want: true,
		},
		{
			name: "net timeout",
			err:  timeoutError{},
			want: true,
		},
		{
			name: "message timeout",
			err:  errString("Client.Timeout exceeded while awaiting headers"),
			want: true,
		},
		{
			name: "non timeout",
			err:  errString("not found"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTimeoutError(tt.err); got != tt.want {
				t.Fatalf("isTimeoutError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

type errString string

func (e errString) Error() string {
	return string(e)
}

type timeoutError struct{}

func (timeoutError) Error() string {
	return "temporary timeout"
}

func (timeoutError) Timeout() bool {
	return true
}

func (timeoutError) Temporary() bool {
	return true
}
