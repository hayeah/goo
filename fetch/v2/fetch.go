package fetch

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/tidwall/gjson"
)

// URLParams map[string]string
// BodyParams any

type Options struct {
	BaseURL    string
	PathParams any

	Method string
	Header http.Header
	Body   string

	Client  *http.Client
	Context context.Context
}

func (o *Options) SetHeader(key, value string) {
	if o.Header == nil {
		o.Header = http.Header{}
	}

	o.Header.Set(key, value)
}

// Do creates a new request and executes it.
func (o *Options) Do(resource string) (*http.Response, error) {
	req, err := NewRequest(resource, o)
	if err != nil {
		return nil, err
	}

	var client *http.Client
	if o.Client != nil {
		client = o.Client
	} else {
		client = http.DefaultClient
	}

	return client.Do(req)
}

func NewRequest(resource string, opts *Options) (*http.Request, error) {
	var err error

	if opts.PathParams != nil {
		resource, err = RenderURLPath(resource, opts.PathParams)
		if err != nil {
			return nil, err
		}
	}

	if opts.BaseURL != "" {
		resource, err = url.JoinPath(opts.BaseURL, resource)
		if err != nil {
			return nil, err
		}
	}

	var ctx context.Context
	if opts.Context != nil {
		ctx = opts.Context
	} else {
		ctx = context.Background()
	}

	var method string
	if opts.Method != "" {
		method = opts.Method
	} else {
		method = http.MethodGet
	}

	body := strings.NewReader(opts.Body)
	req, err := http.NewRequestWithContext(ctx, method, resource, body)
	if err != nil {
		return nil, err
	}

	req.Header = opts.Header
	return req, nil
}

type JSONResponse struct {
	*http.Response

	body []byte
}

// JSON decodes the JSON response from the server.
func (r *JSONResponse) Decode(v interface{}) error {
	return json.Unmarshal(r.body, v)
}

// Body returns the body of the response.
func (r *JSONResponse) Body() []byte {
	return r.body
}

// Get queries json path using GJSON
func (r *JSONResponse) Get(path string) GJSONResult {
	return GJSONResult{gjson.GetBytes(r.body, path)}
}

type GJSONResult struct {
	gjson.Result
}

func (r GJSONResult) Unmarshal(o any) error {
	return json.Unmarshal([]byte(r.Raw), o)
}

// String returns the body of the response as a string.
func (r *JSONResponse) String() string {
	return string(r.body)
}

func JSON(resource string, opts *Options) (*JSONResponse, error) {
	res, err := opts.Do(resource)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return &JSONResponse{
		Response: res,
		body:     body,
	}, nil
}
