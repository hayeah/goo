package fetch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/hayeah/goo"
	"github.com/hayeah/goo/fetch/sse"
	"github.com/tidwall/gjson"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// URLParams map[string]string

type Options struct {
	BaseURL    string
	PathParams any

	QueryParams url.Values

	Header     http.Header
	Body       any // []byte | string
	BodyParams any

	Client  *http.Client
	Context context.Context

	Unmarshal any
	Logger    *slog.Logger
}

// Body returns the body of the request. If the body is a template, it will be rendered.
func (o *Options) RenderBody() ([]byte, error) {
	if o.BodyParams != nil {
		switch body := o.Body.(type) {
		case string:
			return goo.RenderJSON(body, o.BodyParams)
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

// Merge destructively fill in fields of opts with defaults
func (o *Options) Merge(opts *Options) *Options {
	if opts == nil {
		opts = &Options{}
	}

	// Clone the current options
	if opts == nil {
		return o
	}

	// Merge non-empty fields from opts into merged
	if opts.BaseURL == "" {
		opts.BaseURL = o.BaseURL
	}

	if opts.Client == nil {
		opts.Client = o.Client
	}

	if opts.Context == nil {
		opts.Context = o.Context
	}

	if opts.Logger == nil {
		opts.Logger = o.Logger
		if opts.Logger == nil {
			opts.Logger = discardLogger
		}
	}

	if opts.Header != nil {
		for key, values := range o.Header {
			for _, value := range values {
				opts.Header.Add(key, value)
			}
		}
	} else {
		opts.Header = o.Header
	}

	return opts
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
		// not using path.Join because it would escape the query params in the resource path
		resource = strings.TrimRight(opts.BaseURL, "/") + "/" + strings.TrimLeft(resource, "/") + "?" + opts.QueryParams.Encode()
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

	if len(body) > 0 && opts.Header.Get("Content-Type") == "application/json" {
		opts.Logger.Debug("fetch.NewRequest", "body", string(body))
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

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

// Response returns the original http.Response.
func (r *JSONResponse) Response() *http.Response {
	return r.response
}

// JSON decodes the JSON response from the server.
func (r *JSONResponse) Unmarshal(v interface{}) error {
	return json.Unmarshal(r.body, v)
}

// Body returns the body of the response.
func (r *JSONResponse) Body() []byte {
	return r.body
}

// String returns the body of the response as a string.
func (r *JSONResponse) String() string {
	return string(r.body)
}

// Get queries json path using GJSON
func (r *JSONResponse) Get(path string) GJSONResult {
	if path == "" {
		return GJSONResult{gjson.ParseBytes(r.body)}
	}
	return GJSONResult{gjson.GetBytes(r.body, path)}
}

// Pretty returns the body of the response as a pretty-printed string.
func (r *JSONResponse) Pretty() string {
	var buf bytes.Buffer
	err := json.Indent(&buf, r.body, "", "  ")
	if err != nil {
		return r.String()
	}
	return buf.String()
}

type GJSONResult struct {
	gjson.Result
}

func (r GJSONResult) Unmarshal(o any) error {
	return json.Unmarshal([]byte(r.Raw), o)
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

	opts.Logger.Debug("fetch.JSON", "method", method, "url", resource)

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

	if opts.Unmarshal != nil {
		err = json.Unmarshal(body, opts.Unmarshal)

		if err != nil {
			return nil, err
		}
	}

	if res.StatusCode >= 400 {
		opts.Logger.Debug("fetch.JSON error", "body", string(body))
		err = &JSONError{jres}
		return jres, err
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
