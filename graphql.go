package graphql

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/llehouerou/go-graphql-client/pkg/jsonutil"
)

// This function allows you to tweak the HTTP request. It might be useful to set authentication
// headers  amongst other things
type RequestModifier func(*http.Request)

// Client is a GraphQL client.
//
// # Immutable Pattern
//
// The Client's With* methods (WithDebug, WithRequestModifier) follow an
// immutable pattern: they return a new Client instance rather than modifying
// the receiver. This allows for safe concurrent use and makes it clear when
// configuration changes take effect.
//
// Always use the returned Client:
//
//	client = client.WithDebug(true)  // Correct
//	client.WithDebug(true)            // Wrong - original client unchanged
//
// Methods can be chained since each returns a new Client:
//
//	client = client.WithDebug(true).WithRequestModifier(modifier)
//
// Note: This differs from SubscriptionClient, whose With* methods modify
// the receiver and return self (mutable/builder pattern).
type Client struct {
	url             string // GraphQL server URL.
	httpClient      *http.Client
	requestModifier RequestModifier
	debug           bool
}

// NewClient creates a GraphQL client targeting the specified GraphQL server URL.
// If httpClient is nil, then http.DefaultClient is used.
func NewClient(url string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		url:             url,
		httpClient:      httpClient,
		requestModifier: nil,
	}
}

// Query executes a single GraphQL query request,
// with a query derived from q, populating the response into it.
// q should be a pointer to struct that corresponds to the GraphQL schema.
func (c *Client) Query(
	ctx context.Context,
	q any,
	variables any,
	options ...Option,
) error {
	return c.do(ctx, queryOperation, q, variables, options...)
}

// Mutate executes a single GraphQL mutation request,
// with a mutation derived from m, populating the response into it.
// m should be a pointer to struct that corresponds to the GraphQL schema.
func (c *Client) Mutate(
	ctx context.Context,
	m any,
	variables any,
	options ...Option,
) error {
	return c.do(ctx, mutationOperation, m, variables, options...)
}

// Query executes a single GraphQL query request,
// with a query derived from q, populating the response into it.
// q should be a pointer to struct that corresponds to the GraphQL schema.
// return raw bytes message.
func (c *Client) QueryRaw(
	ctx context.Context,
	q any,
	variables any,
	options ...Option,
) ([]byte, error) {
	return c.doRaw(ctx, queryOperation, q, variables, options...)
}

// MutateRaw executes a single GraphQL mutation request,
// with a mutation derived from m, populating the response into it.
// m should be a pointer to struct that corresponds to the GraphQL schema.
// return raw bytes message.
func (c *Client) MutateRaw(
	ctx context.Context,
	m any,
	variables any,
	options ...Option,
) ([]byte, error) {
	return c.doRaw(ctx, mutationOperation, m, variables, options...)
}

// buildAndRequest the common method that builds and send graphql request
func (c *Client) buildAndRequest(
	ctx context.Context,
	op operationType,
	v any,
	variables any,
	options ...Option,
) ([]byte, *http.Response, io.Reader, Errors) {
	var query string
	var err error
	switch op {
	case queryOperation:
		query, err = ConstructQuery(v, variables, options...)
	case mutationOperation:
		query, err = ConstructMutation(v, variables, options...)
	}

	if err != nil {
		return nil, nil, nil, newSimpleErrors(ErrGraphQLEncode, err)
	}

	return c.request(ctx, query, variables)
}

// Request the common method that send graphql request
// handleGzipResponse wraps the response body reader with a gzip decompressor
// if the Content-Encoding header indicates gzip compression.
// Returns the potentially-wrapped reader and any error encountered.
func handleGzipResponse(
	resp *http.Response,
	bodyReader io.Reader,
) (io.ReadCloser, error) {
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gr, err := gzip.NewReader(bodyReader)
		if err != nil {
			return nil, fmt.Errorf("problem trying to create gzip reader: %w", err)
		}
		return gr, nil
	}
	return io.NopCloser(bodyReader), nil
}

// copyResponseForDebug reads the entire response body into memory
// and returns both the bytes and a reader positioned at the start.
// This allows the response to be decoded while preserving a copy for debug logging.
func copyResponseForDebug(r io.Reader) ([]byte, io.Reader, error) {
	respBody, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}
	return respBody, bytes.NewReader(respBody), nil
}

func (c *Client) request(
	ctx context.Context,
	query string,
	variables any,
) ([]byte, *http.Response, io.Reader, Errors) {
	// Build HTTP request with JSON body
	request, reqBody, err := c.BuildRequest(ctx, query, variables)
	if err != nil {
		e := c.NewRequestError(
			ErrRequestError,
			fmt.Errorf("problem constructing request: %w", err),
			request,
			nil,
			bytes.NewReader(reqBody),
			nil,
		)
		return nil, nil, nil, Errors{e}
	}

	// Execute HTTP request
	resp, err := c.httpClient.Do(request)
	if err != nil {
		e := c.NewRequestError(
			ErrRequestError,
			err,
			request,
			nil,
			bytes.NewReader(reqBody),
			nil,
		)
		return nil, nil, nil, Errors{e}
	}
	defer func() { _ = resp.Body.Close() }()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := c.NewRequestError(
			ErrRequestError,
			fmt.Errorf("%v; body: %q", resp.Status, body),
			request,
			nil,
			bytes.NewReader(reqBody),
			nil,
		)
		return nil, nil, nil, Errors{err}
	}

	// Handle gzip decompression
	r, err := handleGzipResponse(resp, resp.Body)
	if err != nil {
		return nil, nil, nil, newSimpleErrors(ErrJsonDecode, err)
	}
	defer func() { _ = r.Close() }()

	// Copy response body for debugging if needed
	var respBody []byte
	var respReader *bytes.Reader
	if c.debug {
		var debugReader io.Reader
		respBody, debugReader, err = copyResponseForDebug(r)
		if err != nil {
			return nil, nil, nil, newSimpleErrors(ErrJsonDecode, err)
		}
		respReader = debugReader.(*bytes.Reader)
		r = io.NopCloser(respReader)
	}

	// Decode GraphQL response
	rawData, gqlErrors := c.DecodeResponse(r)

	if c.debug {
		if respReader != nil {
			_, _ = respReader.Seek(
				0,
				io.SeekStart,
			) // Ignore seek errors for debug logging
		}
	}

	// Handle JSON decode errors
	if len(gqlErrors) > 0 {
		// Check if it's a decode error (has ErrJsonDecode code)
		if code, ok := gqlErrors[0].Extensions["code"].(string); ok && code == ErrJsonDecode {
			we := c.NewRequestError(
				ErrJsonDecode,
				fmt.Errorf("%s", gqlErrors[0].Message),
				request,
				resp,
				bytes.NewReader(reqBody),
				bytes.NewReader(respBody),
			)
			return nil, nil, nil, Errors{we}
		}

		// Handle GraphQL errors - decorate first error if debug mode
		if c.debug &&
			(gqlErrors[0].Extensions == nil || gqlErrors[0].Extensions["request"] == nil) {
			gqlErrors[0] = c.DecorateError(
				gqlErrors[0],
				request,
				resp,
				bytes.NewReader(reqBody),
				bytes.NewReader(respBody),
			)
		}

		return rawData, resp, respReader, gqlErrors
	}

	return rawData, resp, respReader, nil
}

// BuildRequest constructs an HTTP request with JSON body for a GraphQL operation.
// It returns the HTTP request and the request body bytes (useful for error decoration).
func (c *Client) BuildRequest(
	ctx context.Context,
	query string,
	variables any,
) (*http.Request, []byte, error) {
	// Normalize empty variable maps to nil
	if !hasVariables(variables) {
		variables = nil
	}
	in := struct {
		Query     string `json:"query"`
		Variables any    `json:"variables,omitempty"`
	}{
		Query:     query,
		Variables: variables,
	}
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(in)
	if err != nil {
		return nil, nil, err
	}

	reqBody := buf.Bytes()
	reqReader := bytes.NewReader(reqBody)
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.url,
		reqReader,
	)
	if err != nil {
		return nil, reqBody, err
	}
	request.Header.Add("Content-Type", "application/json")

	if c.requestModifier != nil {
		c.requestModifier(request)
	}

	return request, reqBody, nil
}

// ExecuteRequest executes an HTTP request and handles gzip decompression.
// It returns the HTTP response and a reader for the (possibly decompressed) body.
func (c *Client) ExecuteRequest(req *http.Request) (*http.Response, io.Reader, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	r := resp.Body

	// Handle gzip decompression
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gr, err := gzip.NewReader(r)
		if err != nil {
			_ = resp.Body.Close()
			return nil, nil, fmt.Errorf("problem trying to create gzip reader: %w", err)
		}
		// Note: caller is responsible for closing both gr and resp.Body
		r = gr
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(r)
		_ = resp.Body.Close()
		return nil, nil, fmt.Errorf("%v; body: %q", resp.Status, body)
	}

	return resp, r, nil
}

// DecodeResponse decodes a GraphQL JSON response into raw data and errors.
// It returns the raw data bytes (if present) and any GraphQL errors.
func (c *Client) DecodeResponse(reader io.Reader) ([]byte, Errors) {
	var out struct {
		Data   *json.RawMessage
		Errors Errors
	}

	err := json.NewDecoder(reader).Decode(&out)
	if err != nil {
		return nil, newSimpleErrors(ErrJsonDecode, err)
	}

	var rawData []byte
	if out.Data != nil && len(*out.Data) > 0 {
		rawData = *out.Data
	}

	if len(out.Errors) > 0 {
		return rawData, out.Errors
	}

	return rawData, nil
}

// do executes a single GraphQL operation.
// return raw message and error
func (c *Client) doRaw(
	ctx context.Context,
	op operationType,
	v any,
	variables any,
	options ...Option,
) ([]byte, error) {
	data, _, _, err := c.buildAndRequest(ctx, op, v, variables, options...)
	if len(err) > 0 {
		return data, err
	}
	return data, nil
}

// do executes a single GraphQL operation and unmarshal json.
func (c *Client) do(
	ctx context.Context,
	op operationType,
	v any,
	variables any,
	options ...Option,
) error {
	data, resp, respBuf, errs := c.buildAndRequest(
		ctx,
		op,
		v,
		variables,
		options...)
	return c.processResponse(v, data, resp, respBuf, errs)
}

// Executes a pre-built query and unmarshals the response into v. Unlike the Query method you have to specify in the query the
// fields that you want to receive as they are not inferred from v. This method is useful if you need to build the query dynamically.
func (c *Client) Exec(
	ctx context.Context,
	query string,
	v any,
	variables map[string]any,
	options ...Option,
) error {
	data, resp, respBuf, errs := c.request(ctx, query, variables)
	return c.processResponse(v, data, resp, respBuf, errs)
}

// Executes a pre-built query and returns the raw json message. Unlike the Query method you have to specify in the query the
// fields that you want to receive as they are not inferred from the interface. This method is useful if you need to build the query dynamically.
func (c *Client) ExecRaw(
	ctx context.Context,
	query string,
	variables map[string]any,
	options ...Option,
) ([]byte, error) {
	data, _, _, errs := c.request(ctx, query, variables)
	if len(errs) > 0 {
		return data, errs
	}
	return data, nil
}

func (c *Client) processResponse(
	v any,
	data []byte,
	resp *http.Response,
	respBuf io.Reader,
	errs Errors,
) error {
	if len(data) > 0 {
		err := jsonutil.UnmarshalGraphQL(data, v)
		if err != nil {
			we := c.DecorateError(
				newError(ErrGraphQLDecode, err),
				nil,
				resp,
				nil,
				respBuf,
			)
			errs = append(errs, we)
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

// clone creates a copy of the Client with all fields preserved.
// This helper prevents field-copying bugs when adding new fields to Client.
func (c *Client) clone() *Client {
	return &Client{
		url:             c.url,
		httpClient:      c.httpClient,
		requestModifier: c.requestModifier,
		debug:           c.debug,
	}
}

// WithRequestModifier returns a new Client with the request modifier set.
// This allows you to reuse the same TCP connection for multiple slightly
// different requests to the same server (e.g., different authentication
// headers for multitenant applications).
//
// This method follows an immutable pattern: it returns a NEW Client instance
// without modifying the original. You must use the returned Client:
//
//	client = client.WithRequestModifier(modifier)  // Correct
//	client.WithRequestModifier(modifier)            // Wrong - has no effect
//
// The method can be chained with other With* methods:
//
//	client = client.WithRequestModifier(modifier).WithDebug(true)
func (c *Client) WithRequestModifier(f RequestModifier) *Client {
	clone := c.clone()
	clone.requestModifier = f
	return clone
}

// WithDebug returns a new Client with debug mode enabled or disabled.
// When enabled, debug mode adds detailed request/response information to
// error extensions, which is useful for troubleshooting GraphQL API issues.
//
// This method follows an immutable pattern: it returns a NEW Client instance
// without modifying the original. You must use the returned Client:
//
//	client = client.WithDebug(true)  // Correct
//	client.WithDebug(true)            // Wrong - has no effect
//
// The method can be chained with other With* methods:
//
//	client = client.WithDebug(true).WithRequestModifier(modifier)
func (c *Client) WithDebug(debug bool) *Client {
	clone := c.clone()
	clone.debug = debug
	return clone
}

// DecorateError decorates an error with request/response information if debug
// mode is enabled. This helper method centralizes the error decoration logic
// and eliminates repetitive debug checks throughout the codebase.
func (c *Client) DecorateError(
	err Error,
	req *http.Request,
	resp *http.Response,
	reqBody,
	respBody io.Reader,
) Error {
	if !c.debug {
		return err
	}

	if req != nil && reqBody != nil {
		err = err.withRequest(req, reqBody)
	}

	if resp != nil && respBody != nil {
		err = err.withResponse(resp, respBody)
	}

	return err
}

// NewRequestError creates a new error with the given code and decorates it with
// request/response information if debug mode is enabled. This is a convenience
// method that combines error creation and decoration in one step.
func (c *Client) NewRequestError(
	code string,
	err error,
	req *http.Request,
	resp *http.Response,
	reqBody,
	respBody io.Reader,
) Error {
	e := newError(code, err)
	return c.DecorateError(e, req, resp, reqBody, respBody)
}

// errors represents the "errors" array in a response from a GraphQL server.
// If returned via error interface, the slice is expected to contain at least 1 element.
//
// Specification: https://facebook.github.io/graphql/#sec-Errors.
type Errors []Error

type Error struct {
	Message    string         `json:"message"`
	Extensions map[string]any `json:"extensions"`
	Locations  []struct {
		Line   int `json:"line"`
		Column int `json:"column"`
	} `json:"locations"`
}

// RequestInfo contains HTTP request information stored in error extensions.
type RequestInfo struct {
	Headers http.Header
	Body    string
}

// ResponseInfo contains HTTP response information stored in error extensions.
type ResponseInfo struct {
	Headers http.Header
	Body    string
}

// InternalExtensions contains internal debugging information stored in error
// extensions. This information is added when debug mode is enabled.
type InternalExtensions struct {
	Request  *RequestInfo
	Response *ResponseInfo
	Error    error
}

// Error implements error interface.
func (e Error) Error() string {
	return fmt.Sprintf("Message: %s, Locations: %+v", e.Message, e.Locations)
}

// Error implements error interface.
func (e Errors) Error() string {
	b := strings.Builder{}
	for _, err := range e {
		b.WriteString(err.Error())
	}
	return b.String()
}

// GetCode returns the error code from the extensions, or an empty string if
// not present.
func (e Error) GetCode() string {
	if e.Extensions == nil {
		return ""
	}
	code, ok := e.Extensions["code"].(string)
	if !ok {
		return ""
	}
	return code
}

// GetInternalExtensions returns the typed internal extensions, or nil if not
// present. Internal extensions contain debugging information added when debug
// mode is enabled.
func (e Error) GetInternalExtensions() *InternalExtensions {
	if e.Extensions == nil {
		return nil
	}

	internal, ok := e.Extensions["internal"].(map[string]any)
	if !ok {
		return nil
	}

	ext := &InternalExtensions{}

	if req, ok := internal["request"].(map[string]any); ok {
		ext.Request = &RequestInfo{}
		if headers, ok := req["headers"].(http.Header); ok {
			ext.Request.Headers = headers
		}
		if body, ok := req["body"].(string); ok {
			ext.Request.Body = body
		}
	}

	if resp, ok := internal["response"].(map[string]any); ok {
		ext.Response = &ResponseInfo{}
		if headers, ok := resp["headers"].(http.Header); ok {
			ext.Response.Headers = headers
		}
		if body, ok := resp["body"].(string); ok {
			ext.Response.Body = body
		}
	}

	if err, ok := internal["error"].(error); ok {
		ext.Error = err
	}

	return ext
}

func (e Error) getInternalExtension() map[string]any {
	if e.Extensions == nil {
		return make(map[string]any)
	}

	if ex, ok := e.Extensions["internal"]; ok {
		return ex.(map[string]any)
	}

	return make(map[string]any)
}

// newError creates a new Error with the given code and underlying error.
// The underlying error is stored in the extensions for debugging.
func newError(code string, err error) Error {
	return Error{
		Message: err.Error(),
		Extensions: map[string]any{
			"code": code,
		},
	}
}

// newSimpleErrors creates an Errors slice with a single error, wrapping the
// given error with the specified code. This is a convenience method for simple
// error cases that don't have request/response context.
func newSimpleErrors(code string, err error) Errors {
	return Errors{newError(code, err)}
}

// withDebugInfo adds debug information to the error's internal extensions.
// It reads the body from bodyReader and stores it along with headers under the
// specified infoType key ("request" or "response").
func (e Error) withDebugInfo(
	infoType string,
	headers http.Header,
	bodyReader io.Reader,
) Error {
	internal := e.getInternalExtension()
	bodyBytes, err := io.ReadAll(bodyReader)
	if err != nil {
		internal["error"] = err
	} else {
		internal[infoType] = map[string]any{
			"headers": headers,
			"body":    string(bodyBytes),
		}
	}

	if e.Extensions == nil {
		e.Extensions = make(map[string]any)
	}
	e.Extensions["internal"] = internal
	return e
}

func (e Error) withRequest(req *http.Request, bodyReader io.Reader) Error {
	return e.withDebugInfo("request", req.Header, bodyReader)
}

func (e Error) withResponse(res *http.Response, bodyReader io.Reader) Error {
	return e.withDebugInfo("response", res.Header, bodyReader)
}

// UnmarshalGraphQL parses the JSON-encoded GraphQL response data and stores
// the result in the GraphQL query data structure pointed to by v.
//
// The implementation is created on top of the JSON tokenizer available
// in "encoding/json".Decoder.
// This function is re-exported from the internal package
func UnmarshalGraphQL(data []byte, v any) error {
	return jsonutil.UnmarshalGraphQL(data, v)
}

type operationType uint8

const (
	queryOperation operationType = iota
	mutationOperation
	// subscriptionOperation // Unused.
)

const (
	ErrRequestError  = "request_error"
	ErrJsonEncode    = "json_encode_error"
	ErrJsonDecode    = "json_decode_error"
	ErrGraphQLEncode = "graphql_encode_error"
	ErrGraphQLDecode = "graphql_decode_error"
)
