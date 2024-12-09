package fetch_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hayeah/goo/fetch"
)

func TestOptionsCloneAndMerge(t *testing.T) {
	tests := []struct {
		name     string
		original *fetch.Options
		merge    *fetch.Options
		expected *fetch.Options
	}{
		{
			name: "Merge non-empty fields",
			original: &fetch.Options{
				BaseURL: "https://original.com",
				Header:  http.Header{"X-Original-Header": {"original-value"}},
			},
			merge: &fetch.Options{
				BaseURL: "https://merged.com",
				Header:  http.Header{"X-Merged-Header": {"merged-value"}},
			},
			expected: &fetch.Options{
				BaseURL: "https://merged.com",
				Header:  http.Header{"X-Original-Header": {"original-value"}, "X-Merged-Header": {"merged-value"}},
			},
		},
		{
			name: "Merge with empty merge options",
			original: &fetch.Options{
				BaseURL: "https://original.com",
			},
			merge: &fetch.Options{},
			expected: &fetch.Options{
				BaseURL: "https://original.com",
			},
		},
		{
			name: "Merge nil headers",
			original: &fetch.Options{
				BaseURL: "https://original.com",
			},
			merge: &fetch.Options{
				BaseURL: "https://merged.com",
				Header:  nil,
			},
			expected: &fetch.Options{
				BaseURL: "https://merged.com",
			},
		},
		{
			name: "Merge body and body params",
			original: &fetch.Options{
				Body:       "original body",
				BodyParams: map[string]string{"original": "param"},
			},
			merge: &fetch.Options{
				Body:       "merged body",
				BodyParams: map[string]string{"merged": "param"},
			},
			expected: &fetch.Options{
				Body:       "merged body",
				BodyParams: map[string]string{"merged": "param"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			// Test Clone
			clone := tc.original.Clone()
			assert.Equal(tc.original, clone, "Cloned options should be equal to original")

			// Test Merge
			merged := tc.original.Merge(tc.merge)
			assert.Equal(tc.expected, merged, "Merged options should match expected options")
		})
	}
}

func TestRenderBody(t *testing.T) {
	assert := assert.New(t)

	t.Run("with nil BodyParams and nil Body", func(t *testing.T) {
		opts := fetch.Options{}
		body, err := opts.RenderBody()
		assert.NoError(err)
		assert.Nil(body)
	})

	t.Run("with string Body and nil BodyParams", func(t *testing.T) {
		opts := fetch.Options{
			Body: "test body",
		}
		body, err := opts.RenderBody()
		assert.NoError(err)
		assert.Equal([]byte("test body"), body)
	})

	t.Run("with []byte Body and nil BodyParams", func(t *testing.T) {
		opts := fetch.Options{
			Body: []byte("test body"),
		}
		body, err := opts.RenderBody()
		assert.NoError(err)
		assert.Equal([]byte("test body"), body)
	})

	t.Run("with struct Body and nil BodyParams", func(t *testing.T) {
		type RequestBody struct {
			Key string `json:"key"`
		}

		opts := fetch.Options{
			Body: RequestBody{Key: "value"},
		}
		body, err := opts.RenderBody()
		assert.NoError(err)
		assert.JSONEq(`{"key":"value"}`, string(body))
	})

	t.Run("with string Body and BodyParams", func(t *testing.T) {
		opts := fetch.Options{
			Body:       `{"key": {{Key}}}`,
			BodyParams: map[string]string{"Key": "value"},
		}
		body, err := opts.RenderBody()
		assert.NoError(err)
		assert.JSONEq(`{"key":"value"}`, string(body))
	})

	t.Run("with string BodyParams but non-string Body", func(t *testing.T) {
		opts := fetch.Options{
			Body:       123,
			BodyParams: map[string]string{"Key": "value"},
		}
		body, err := opts.RenderBody()
		assert.Error(err)
		assert.Nil(body)
	})
}

func TestNewRequest_RenderBodyIntegration(t *testing.T) {
	assert := assert.New(t)

	t.Run("with valid Body and BodyParams", func(t *testing.T) {
		opts := &fetch.Options{
			Body:       `{"key": {{Key}} }`,
			BodyParams: map[string]string{"Key": "value"},
			Header:     http.Header{"Content-Type": {"application/json"}},
		}

		req, err := fetch.NewRequest("POST", "http://example.com/api/v1/resource", opts)
		assert.NoError(err)
		assert.NotNil(req)
		assert.Equal("http://example.com/api/v1/resource", req.URL.String())
		assert.Equal(http.MethodPost, req.Method)
		assert.Equal("application/json", req.Header.Get("Content-Type"))
		body, _ := io.ReadAll(req.Body)
		assert.JSONEq(`{"key":"value"}`, string(body))
	})

	t.Run("with invalid BodyParams", func(t *testing.T) {
		opts := &fetch.Options{
			Body:       123,
			BodyParams: map[string]string{"Key": "value"},
			Header:     http.Header{"Content-Type": {"application/json"}},
		}

		req, err := fetch.NewRequest(http.MethodPost, "http://example.com/api/v1/resource", opts)
		assert.Error(err)
		assert.Nil(req)
	})

	t.Run("with nil Body and BodyParams", func(t *testing.T) {
		opts := &fetch.Options{
			Header: http.Header{"Content-Type": {"application/json"}},
		}

		req, err := fetch.NewRequest(http.MethodPost, "http://example.com/api/v1/resource", opts)
		assert.NoError(err)
		assert.NotNil(req)
		assert.Equal("http://example.com/api/v1/resource", req.URL.String())
		assert.Equal(http.MethodPost, req.Method)
		assert.Equal("application/json", req.Header.Get("Content-Type"))
		assert.Nil(req.Body)
	})
}

var commonOpts = &fetch.Options{
	BaseURL: "http://example.com",
	Client:  &http.Client{},
	Context: context.Background(),
}

func TestNewRequest(t *testing.T) {
	assert := assert.New(t)

	t.Run("with baseURL and URL", func(t *testing.T) {
		opts := *commonOpts
		opts.SetHeader("Content-Type", "application/json")

		req, err := fetch.NewRequest(http.MethodGet, "/api/v1/resource", &opts)
		assert.NoError(err)
		assert.Equal("http://example.com/api/v1/resource", req.URL.String())
		assert.Equal(http.MethodGet, req.Method)
		assert.Equal("application/json", req.Header.Get("Content-Type"))
	})

	t.Run("with PathParams", func(t *testing.T) {
		opts := *commonOpts
		opts.PathParams = map[string]interface{}{"key1": "value1", "key2": "value2"}

		req, err := fetch.NewRequest(http.MethodGet, "/api/v1/{{key1}}/{{key2}}", &opts)
		assert.NoError(err)
		assert.Equal("http://example.com/api/v1/value1/value2", req.URL.String())
		assert.Equal(http.MethodGet, req.Method)
	})

	t.Run("with PathParams struct", func(t *testing.T) {
		opts := *commonOpts
		type PathParams struct {
			Key1 string
			Key2 string
		}

		opts.PathParams = PathParams{"value1", "value2"}

		req, err := fetch.NewRequest(http.MethodGet, "/api/v1/{{Key1}}/{{Key2}}", &opts)
		assert.NoError(err)
		assert.Equal("http://example.com/api/v1/value1/value2", req.URL.String())
		assert.Equal(http.MethodGet, req.Method)
	})

	t.Run("without baseURL", func(t *testing.T) {
		opts := &fetch.Options{
			Header: http.Header{"Content-Type": {"application/json"}},
			Body:   `{"key":"value"}`,
		}

		req, err := fetch.NewRequest(http.MethodPost, "http://example.com/api/v1/resource", opts)
		assert.NoError(err)
		assert.Equal("http://example.com/api/v1/resource", req.URL.String())
		assert.Equal(http.MethodPost, req.Method)
		assert.Equal("application/json", req.Header.Get("Content-Type"))
		body, _ := io.ReadAll(req.Body)
		assert.Equal(`{"key":"value"}`, string(body))
	})

	t.Run("with context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		opts := *commonOpts
		opts.Context = ctx

		req, err := fetch.NewRequest(http.MethodGet, "/api/v1/resource", &opts)
		assert.NoError(err)
		assert.Equal(ctx, req.Context())
	})

	t.Run("without context", func(t *testing.T) {
		opts := *commonOpts
		opts.Context = nil

		req, err := fetch.NewRequest(http.MethodGet, "/api/v1/resource", &opts)
		assert.NoError(err)
		assert.NotNil(req.Context())
		assert.Equal(context.Background(), req.Context())
	})

	t.Run("invalid baseURL and URL", func(t *testing.T) {
		opts := *commonOpts
		opts.BaseURL = ":/invalid-url"

		req, err := fetch.NewRequest(http.MethodGet, "/api/v1/resource", &opts)
		assert.Error(err)
		assert.Nil(req)
	})
}

func TestJSON(t *testing.T) {
	assert := assert.New(t)

	type TestData struct {
		Data int
	}

	type TestError struct {
		Error string
	}

	tests := []struct {
		name           string
		handlerFunc    http.HandlerFunc
		options        *fetch.Options
		wantErr        bool
		expectedBody   string
		expectedDecode any
	}{
		{
			name: "successful request",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"data": 42 }`))
			},
			options:        &fetch.Options{},
			wantErr:        false,
			expectedBody:   `{"data": 42 }`,
			expectedDecode: TestData{},
		},
		{
			name: "404 not found",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error": "Not Found"}`))
			},
			options:        &fetch.Options{},
			wantErr:        true,
			expectedBody:   `{"error": "Not Found"}`,
			expectedDecode: TestError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.handlerFunc))
			defer server.Close()

			tt.options.BaseURL = server.URL

			resp, err := fetch.JSON(http.MethodGet, "/test", tt.options)
			if tt.wantErr {
				assert.Error(err)

				jsonErr, ok := err.(*fetch.JSONError)
				assert.True(ok)

				assert.JSONEq(tt.expectedBody, string(jsonErr.Body()))
			} else {
				assert.NoError(err)
				assert.JSONEq(tt.expectedBody, string(resp.Body()))
				assert.JSONEq(tt.expectedBody, resp.String())

				err = resp.Unmarshal(&tt.expectedDecode)
				assert.NoError(err)

				decodedValue, err := json.Marshal(&tt.expectedDecode)
				assert.NoError(err)

				assert.JSONEq(tt.expectedBody, string(decodedValue))

			}
		})
	}
}
