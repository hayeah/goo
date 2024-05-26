package fetch

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

type HTTPError struct {
	StatusCode int
	Status     string
	Body       []byte
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("http error: %s", e.Status)
}

func New(c *resty.Client) *Client {
	return &Client{c}
}

type Client struct {
	*resty.Client
}

// R returns a GJSONRequest
func (c *Client) R() *Request {
	return &Request{c.Client.R()}
}

// Get
func (c *Client) Get(url string, opts *Options) (*Response, error) {
	return c.R().JSON(http.MethodGet, url, opts)
}

// Post sends a POST request
func (c *Client) Post(url string, opts *Options) (*Response, error) {
	return c.R().JSON(http.MethodPost, url, opts)
}

type Options struct {
	Method string

	Body       any
	BodyParams any
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
	} else if o.Body == nil {
		return json.Marshal(o.Body)
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

	body, err := opts.RenderBody()
	if err != nil {
		return nil, err
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
	if res.IsError() {
		err = &HTTPError{
			StatusCode: res.StatusCode(),
			Status:     res.Status(),
			Body:       res.Body(),
		}
	}

	return &Response{res}, err
}

type Response struct {
	*resty.Response
}

func (r *Response) JSON() []byte {
	// TODO: check json content type
	return r.Body()
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
