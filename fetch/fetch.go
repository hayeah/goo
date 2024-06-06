package fetch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/tidwall/gjson"
)

// URLParams map[string]string

type Options struct {
	BaseURL    string
	PathParams any

	Method     string
	Header     http.Header
	Body       any // []byte | string
	BodyParams any

	Client  *http.Client
	Context context.Context
}

// Body returns the body of the request. If the body is a template, it will be rendered.
func (o *Options) RenderBody() ([]byte, error) {
	if o.BodyParams != nil {
		switch body := o.Body.(type) {
		case string:
			return RenderJSON(body, o.BodyParams)
		default:
			return nil, errors.New("body should be a string template")
		}
	} else if o.Body != nil {
		switch body := o.Body.(type) {
		case string:
			return []byte(body), nil
		case []byte:
			return body, nil
		default:
			return json.Marshal(o.Body)
		}

	}

	return nil, nil
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

	body, err := opts.RenderBody()
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = io.NopCloser(bytes.NewReader(body))
	}

	req, err := http.NewRequestWithContext(ctx, method, resource, bodyReader)
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
	// set content type to json is not set
	// if opts.Header.Get("Content-Type") == "" {
	// 	opts.SetHeader("Content-Type", "application/json")
	// }

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
