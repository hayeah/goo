package fetch_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hayeah/goo/fetch/v2"
)

var commonOpts = &fetch.Options{
	BaseURL: "http://example.com",
	Client:  &http.Client{},
	Context: context.Background(),
}

func TestNewRequest(t *testing.T) {
	assert := assert.New(t)

	t.Run("with baseURL and URL", func(t *testing.T) {
		opts := *commonOpts
		opts.Method = http.MethodGet
		opts.SetHeader("Content-Type", "application/json")

		req, err := fetch.NewRequest("/api/v1/resource", &opts)
		assert.NoError(err)
		assert.Equal("http://example.com/api/v1/resource", req.URL.String())
		assert.Equal(http.MethodGet, req.Method)
		assert.Equal("application/json", req.Header.Get("Content-Type"))
	})

	t.Run("with PathParams", func(t *testing.T) {
		opts := *commonOpts
		opts.PathParams = map[string]interface{}{"key1": "value1", "key2": "value2"}

		req, err := fetch.NewRequest("/api/v1/{{key1}}/{{key2}}", &opts)
		assert.NoError(err)
		assert.Equal("http://example.com/api/v1/value1/value2", req.URL.String())
		assert.Equal(http.MethodGet, req.Method)
	})

	t.Run("with PathParams struct", func(t *testing.T) {
		opts := *commonOpts
		opts.Method = http.MethodGet
		type PathParams struct {
			Key1 string
			Key2 string
		}

		opts.PathParams = PathParams{"value1", "value2"}

		req, err := fetch.NewRequest("/api/v1/{{Key1}}/{{Key2}}", &opts)
		assert.NoError(err)
		assert.Equal("http://example.com/api/v1/value1/value2", req.URL.String())
		assert.Equal(http.MethodGet, req.Method)
	})

	t.Run("without baseURL", func(t *testing.T) {
		opts := &fetch.Options{
			Method: http.MethodPost,
			Header: http.Header{"Content-Type": {"application/json"}},
			Body:   `{"key":"value"}`,
		}

		req, err := fetch.NewRequest("http://example.com/api/v1/resource", opts)
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
		opts.Method = http.MethodGet
		opts.Context = ctx

		req, err := fetch.NewRequest("/api/v1/resource", &opts)
		assert.NoError(err)
		assert.Equal(ctx, req.Context())
	})

	t.Run("without context", func(t *testing.T) {
		opts := *commonOpts
		opts.Method = http.MethodGet
		opts.Context = nil

		req, err := fetch.NewRequest("/api/v1/resource", &opts)
		assert.NoError(err)
		assert.NotNil(req.Context())
		assert.Equal(context.Background(), req.Context())
	})

	t.Run("invalid baseURL and URL", func(t *testing.T) {
		opts := *commonOpts
		opts.BaseURL = ":/invalid-url"
		opts.Method = http.MethodGet

		req, err := fetch.NewRequest("/api/v1/resource", &opts)
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
			options: &fetch.Options{
				Method: http.MethodGet,
			},
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
			options: &fetch.Options{
				Method: http.MethodGet,
			},
			wantErr:        false,
			expectedBody:   `{"error": "Not Found"}`,
			expectedDecode: TestError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.handlerFunc))
			defer server.Close()

			tt.options.BaseURL = server.URL

			resp, err := fetch.JSON("/test", tt.options)
			if tt.wantErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.JSONEq(tt.expectedBody, string(resp.Body()))
				assert.JSONEq(tt.expectedBody, resp.String())

				err = resp.Decode(&tt.expectedDecode)
				assert.NoError(err)

				decodedValue, err := json.Marshal(&tt.expectedDecode)
				assert.NoError(err)

				assert.JSONEq(tt.expectedBody, string(decodedValue))

			}
		})
	}
}
