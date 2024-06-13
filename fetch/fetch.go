package fetch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/hayeah/goo/sse"
	"github.com/tidwall/gjson"
)

// URLParams map[string]string

type Options struct {
	BaseURL    string
	PathParams any

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
func (o *Options) Do(method, resource string) (*http.Response, error) {
	method = strings.ToUpper(method)

	req, err := NewRequest(method, resource, o)
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

// JSON creates a new request and executes it as a JSON request.
func (o *Options) JSON(method, resource string, opts *Options) (*JSONResponse, error) {
	opts2 := o.Merge(opts)
	return JSON(method, resource, opts2)
}

// SSE creates a new request and executes it as an SSE request.
func (o *Options) SSE(method, resource string, opts *Options) (*SSEResponse, error) {
	opts2 := o.Merge(opts)
	return SSE(method, resource, opts2)
}

// Clone creates a deep copy of the Options.
func (o *Options) Clone() *Options {
	// Create a new Options struct and copy the basic fields
	clone := &Options{
		BaseURL:    o.BaseURL,
		PathParams: o.PathParams,
		Header:     o.Header.Clone(), // Deep copy of headers
		Body:       o.Body,
		BodyParams: o.BodyParams,
		Client:     o.Client,
		Context:    o.Context,
	}

	return clone
}

// Merge merges the options with another set of options.
func (o *Options) Merge(opts *Options) *Options {
	// Clone the current options
	merged := o.Clone()

	if opts == nil {
		return o
	}

	// Merge non-empty fields from opts into merged
	if opts.BaseURL != "" {
		merged.BaseURL = opts.BaseURL
	}
	if opts.PathParams != nil {
		merged.PathParams = opts.PathParams
	}

	if opts.Header != nil {
		for key, values := range opts.Header {
			for _, value := range values {
				merged.Header.Add(key, value)
			}
		}
	}

	if opts.Body != nil {
		merged.Body = opts.Body
	}
	if opts.BodyParams != nil {
		merged.BodyParams = opts.BodyParams
	}
	if opts.Client != nil {
		merged.Client = opts.Client
	}
	if opts.Context != nil {
		merged.Context = opts.Context
	}

	return merged
}
func NewRequest(method, resource string, opts *Options) (*http.Request, error) {
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

	body, err := opts.RenderBody()
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	// fmt.Printf("%s %s\n", method, resource)

	req, err := http.NewRequestWithContext(ctx, method, resource, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header = opts.Header
	return req, nil
}

type JSONResponse struct {
	response *http.Response

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

type JSONError struct {
	// StatusCode int
	// Status     string
	// Body     []byte
	// Response *http.Response
	*JSONResponse
}

func (e *JSONError) Error() string {
	return fmt.Sprintf("fetch JSON error: %d %s", e.response.StatusCode, e.response.Status)
}

// String returns the body of the response as a string.
func (e *JSONError) String() string {
	return string(e.body)
}

func JSON(method, resource string, opts *Options) (*JSONResponse, error) {
	// set content type to json is not set
	// if opts.Header.Get("Content-Type") == "" {
	// 	opts.SetHeader("Content-Type", "application/json")
	// }
	res, err := opts.Do(method, resource)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	jres := &JSONResponse{
		response: res,
		body:     body,
	}

	if res.StatusCode >= 400 {
		err = &JSONError{jres}
		return nil, err
	}

	return jres, nil

}

type SSEResponse struct {
	*sse.Scanner
}

func (r *SSEResponse) Close() error {
	return r.Scanner.Close()
}

func SSE(method, resource string, opts *Options) (*SSEResponse, error) {
	res, err := opts.Do(method, resource)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 400 {
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		jres := &JSONResponse{response: res, body: body}
		return nil, &JSONError{jres}
	}

	scanner := sse.NewScanner(res.Body, false)
	return &SSEResponse{scanner}, nil
}
