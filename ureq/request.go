package ureq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/volodymyrprokopyuk/go-util/udump"
)

const (
  AuthZHeader = "Authorization"
  AuthZBearer = "Bearer "
)

const (
  contentType = "Content-Type"
  appJSON = "application/json"
  appForm = "application/x-www-form-urlencoded"
)

func ReadJSON[T any](data []byte) (*T, error) {
  var val T
  err := json.Unmarshal(data, &val)
  if err != nil {
    return nil, err
  }
  return &val, nil
}

func BoolP(b bool) *bool {
  return &b
}

func IntP(i int) *int {
  return &i
}

func StringP(s string) *string {
  return &s
}

func TimeP(t time.Time) *time.Time {
  return &t
}

type Client struct {
  client *http.Client
  baseURL string
}

type clientConfig struct {
  baseURL string
  timeout time.Duration
  keepAlive bool
}

type clientOption func (cfg *clientConfig)

func BaseURL(baseURL string) clientOption {
  return func(cfg *clientConfig) {
    cfg.baseURL = baseURL
  }
}

func Timeout(timeout time.Duration) clientOption {
  return func (cfg *clientConfig) {
    cfg.timeout = timeout
  }
}

func KeepAlive(keepAlive bool) clientOption {
  return func(cfg *clientConfig) {
    cfg.keepAlive = keepAlive
  }
}

func NewClient(opts ...clientOption) *Client {
  cfg := &clientConfig{
    timeout: 5 * time.Second,
    keepAlive: true,
  }
  for _, opt := range opts {
    opt(cfg)
  }
  trn := &http.Transport{
    DisableKeepAlives: !cfg.keepAlive,
  }
  cln := &http.Client{
    Transport: trn,
    Timeout: cfg.timeout,
  }
  return &Client{
    client: cln,
    baseURL: cfg.baseURL,
  }
}

type requestConfig struct {
  err error
  trace bool
  url string
  query map[string]string
  header map[string]string
  reqBytes []byte
  resValue any
  resError any
  resBytes *[]byte
}

type requestOption func (cfg *requestConfig)

func Trace() requestOption {
  return func (cfg *requestConfig) {
    cfg.trace = true
  }
}

func URL(val string) requestOption {
  return func(cfg *requestConfig) {
    cfg.url = val
  }
}

func Query(key, value string) requestOption {
  return func(cfg *requestConfig) {
    cfg.query[key] = value
  }
}

func Header(key, value string) requestOption {
  return func(cfg *requestConfig) {
    cfg.header[key] = value
  }
}

func Bearer(tok string) requestOption {
  return func(cfg *requestConfig) {
    cfg.header[AuthZHeader] = AuthZBearer + tok
  }
}

func FormValues(form url.Values) requestOption {
  return func(cfg *requestConfig) {
    cfg.reqBytes = []byte(form.Encode())
    cfg.header[contentType] = appForm
  }
}

func ReqJSON(value any) requestOption {
  return func(cfg *requestConfig) {
    jvalue, err := json.Marshal(value)
    if err != nil {
      cfg.err = err
      return
    }
    cfg.reqBytes = jvalue
    cfg.header[contentType] = appJSON
  }
}

func ResJSON(value any) requestOption {
  return func(cfg *requestConfig) {
    cfg.resValue = value
  }
}

func ErrJSON(value any) requestOption {
  return func(cfg *requestConfig) {
    cfg.resError = value
  }
}

func ReqBytes(value []byte) requestOption {
  return func(cfg *requestConfig) {
    cfg.reqBytes = value
  }
}

func ResBytes(value *[]byte) requestOption {
  return func(cfg *requestConfig) {
    cfg.resBytes = value
  }
}

func traceReq(method string, cfg *requestConfig) {
  // HTTP method and URL
  fmt.Printf("%s %s\n", method, cfg.url)
  // Query
  if len(cfg.query) > 0 {
    fmt.Printf("query %s\n", udump.Value(cfg.query))
  }
  // Headers
  var contType string
  for key, value := range cfg.header {
    if key == contentType {
      contType = value
      continue
    }
    fmt.Printf(">> %s: %s\n", key, value)
  }
  // Body
  if len(cfg.reqBytes) > 0 {
    if contType == appJSON {
      fmt.Printf(">> %s\n", udump.JSON(cfg.reqBytes))
    } else {
      fmt.Printf(">> %s\n", cfg.reqBytes)
    }
  }
}

func traceRes(res *http.Response, body []byte, start time.Time) {
  elapsed := time.Since(start).Truncate(time.Millisecond)
  if len(body) > 0 {
    if res.Header.Get(contentType) == appJSON {
      fmt.Printf("<< %d %s %s\n", res.StatusCode, elapsed, udump.JSON(body))
    } else {
      fmt.Printf("<< %d %s %s\n", res.StatusCode, elapsed, body)
    }
  } else {
    fmt.Printf("<< %d %s\n", res.StatusCode, elapsed)
  }
}

func (c *Client) request(
  ctx context.Context, method string, opts ...requestOption,
) (*http.Response, error) {
  // Process request configuration options
  success := []int{200, 201, 202, 204}
  cfg := &requestConfig{
    query: make(map[string]string),
    header: make(map[string]string),
  }
  for _, opt := range opts {
    opt(cfg)
    if cfg.err != nil {
      return nil, cfg.err
    }
  }
  // URL
  if len(c.baseURL) == 0 && len(cfg.url) == 0 {
    return nil, fmt.Errorf("%s empty request URL", method)
  }
  url2 := c.baseURL + cfg.url
  // Create a request
  req, err := http.NewRequestWithContext(
    ctx, method, url2, bytes.NewReader(cfg.reqBytes),
  )
  if err != nil {
    return nil, err
  }
  // Query
  query := req.URL.Query()
  for key, value := range cfg.query {
    query.Set(key, value)
  }
  req.URL.RawQuery = query.Encode()
  // Header
  for key, value := range cfg.header {
    req.Header.Set(key, value)
  }
  var start time.Time
  if cfg.trace {
    traceReq(method, cfg)
    start = time.Now()
  }
  // Perform a request
  res, err := c.client.Do(req)
  if err != nil {
    return nil, err
  }
  defer func() {
    _ = res.Body.Close()
  }()
  body, err := io.ReadAll(res.Body)
  if err != nil {
    return nil, err
  }
  if cfg.trace {
    traceRes(res, body, start)
  }
  // Valid response
  if slices.Contains(success, res.StatusCode) && cfg.resValue != nil {
    err = json.Unmarshal(body, cfg.resValue)
    if err != nil {
      return nil, err
    }
    return res, nil
  }
  // Error response
  if !slices.Contains(success, res.StatusCode) && cfg.resError != nil {
    err = json.Unmarshal(body, cfg.resError)
    if err != nil {
      return nil, err
    }
  }
  // Response bytes
  if cfg.resBytes != nil {
    *cfg.resBytes = body
  }
  return res, nil
}

func (c *Client) GET(
  ctx context.Context, opts ...requestOption,
) (*http.Response, error) {
  return c.request(ctx, http.MethodGet, opts...)
}

func (c *Client) POST(
  ctx context.Context, opts ...requestOption,
) (*http.Response, error) {
  return c.request(ctx, http.MethodPost, opts...)
}

func (c *Client) PUT(
  ctx context.Context, opts ...requestOption,
) (*http.Response, error) {
  return c.request(ctx, http.MethodPut, opts...)
}

func (c *Client) PATCH(
  ctx context.Context, opts ...requestOption,
) (*http.Response, error) {
  return c.request(ctx, http.MethodPatch, opts...)
}

func (c *Client) FORM(
  ctx context.Context, opts ...requestOption,
) (*http.Response, error) {
  return c.request(ctx, http.MethodPost, opts...)
}
