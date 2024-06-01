package fetch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/go-resty/resty/v2"
	"github.com/hayeah/goo/sse"
	"github.com/tidwall/gjson"
)

type HTTPError struct {
	StatusCode int
	Status     string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("http error: %s", e.Status)
}

func New(c *resty.Client) Client {
	return Client{c}
}

type Client struct {
	*resty.Client
}

// R returns a GJSONRequest
func (c *Client) R() *Request {
	return &Request{c.Client.R()}
}

func (c *Client) JSON(method, url string, opts *Options) (*Response, error) {
	return c.R().JSON(method, url, opts)
}

type SSEResponse struct {
	*Response
	*sse.Scanner
}

func (r *SSEResponse) Close() error {
	return r.Scanner.Close()
}

func (c *Client) SSE(method string, url string, opts *Options) (*SSEResponse, error) {

	var opts2 Options
	if opts != nil {
		opts2 = *opts
	}

	opts2.RawResponseBody = true

	res, err := c.R().JSON(method, url, &opts2)
	if err != nil {
		return nil, err
	}

	scanner := sse.NewScanner(res.RawBody(), false)

	return &SSEResponse{res, scanner}, nil
}

type Options struct {
	Method string

	Body       any
	BodyParams any

	RawResponseBody bool

	PathParams any // map[string]string
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

type Request struct {
	*resty.Request
}

func (r *Request) JSON(method, url string, opts *Options) (*Response, error) {
	if opts == nil {
		opts = &Options{}
	}

	if opts.RawResponseBody {
		r.SetDoNotParseResponse(true)
	}

	body, err := opts.RenderBody()
	if err != nil {
		return nil, err
	}

	if opts.PathParams != nil {
		switch opts.PathParams.(type) {
		case map[string]string:
			r.SetPathParams(opts.PathParams.(map[string]string))
		default:
			// TODO: use mapstructure to convert to map[string]string
			// github.com/mitchellh/mapstructure
			return nil, errors.New("path params should be map[string]string")
		}

	}

	r.SetHeader("Content-Type", "application/json")
	if body != nil {
		r.SetBody(body)
	}

	// method := "GET"
	// if opts.Method != "" {
	// 	method = opts.Method
	// }

	res, err := r.Execute(method, url)
	if err != nil {
		return nil, err
	}

	// return http status error
	// if res.IsError() {
	// 	err = &HTTPError{
	// 		StatusCode: res.StatusCode(),
	// 		Status:     res.Status(),
	// 	}
	// }

	return &Response{res}, err
}

type Response struct {
	*resty.Response
}

func (r *Response) JSON() []byte {
	// TODO: check json content type
	return r.Body()
}

func (r *Response) String() string {
	//
	body := r.Body()
	if body == nil {
		//
		data, err := io.ReadAll(r.RawBody())

		if err != nil {
			r.Request.SetError(err)
		}
		// r.SetBody(data)

		// r.Request.SetError(errors.New("body is nil")
		return string(data)
	}

	return string(r.Body())
}

func (r *Response) Close() error {
	if r != nil {
		return r.Response.RawBody().Close()
	}

	return nil
}

// GJSON queries body
func (r *Response) GJSON(path string) GJSONResult {
	return GJSONResult{gjson.GetBytes(r.Body(), path)}
}

type GJSONResult struct {
	gjson.Result
}

func (r GJSONResult) Unmarshal(o any) error {
	return json.Unmarshal([]byte(r.Raw), o)
}
