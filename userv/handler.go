package userv

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/volodymyrprokopyuk/go-util/udump"
)

type BadRequest string // 400

func (e BadRequest) Error() string {
  return string(e)
}

type Unautorized string // 401

func (e Unautorized) Error() string {
  return string(e)
}

type Forbidden string // 403

func (e Forbidden) Error() string {
  return string(e)
}

type NotFound string // 404

func (e NotFound) Error() string {
  return string(e)
}

type InternalServerError string // 500

func (e InternalServerError) Error() string {
  return string(e)
}

type NotImplemented string // 501

func (e NotImplemented) Error() string {
  return string(e)
}

type ServiceUnavailable string // 503

func (e ServiceUnavailable) Error() string {
  return string(e)
}

func errorStatusCode(err error) int {
  var badRequest BadRequest
  var unauthorized Unautorized
  var forbidden Forbidden
  var notFound NotFound
  var notImplemented NotImplemented
  var serviceUnavailable ServiceUnavailable
  switch {
  case errors.As(err, &badRequest):
    return http.StatusBadRequest
  case errors.As(err, &unauthorized):
    return http.StatusUnauthorized
  case errors.As(err, &forbidden):
    return http.StatusForbidden
  case errors.As(err, &notFound):
    return http.StatusNotFound
  case errors.As(err, &notImplemented):
    return http.StatusNotImplemented
  case errors.As(err, &serviceUnavailable):
    return http.StatusServiceUnavailable
  default:
    return http.StatusInternalServerError
  }
}

func ReadBody[T any](r *http.Request) (*T, error) {
  defer func() {
    _ = r.Body.Close()
  }()
  var val T
  limited := io.LimitReader(r.Body, 10 << 20) // max 10 MB
  body, err := io.ReadAll(limited)
  if err != nil {
    return nil, BadRequest(err.Error())
  }
  err = json.Unmarshal(body, &val)
  if err != nil {
    return nil, BadRequest(err.Error())
  }
  return &val, nil
}

type resError struct {
  Error string `json:"error"`
}

func WriteResponse(w http.ResponseWriter, statusCode int, res any) {
  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(statusCode)
  if res != nil {
    jres, _ := json.Marshal(res)
    _, _ = w.Write(jres)
  }
}

func WriteError(w http.ResponseWriter, err error) {
  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(errorStatusCode(err))
  res := resError{Error: err.Error()}
  jres, _ := json.Marshal(res)
  _, _ = w.Write(jres)
}

type Middleware func(next http.HandlerFunc) http.HandlerFunc

func NotFoundHandler(mux *http.ServeMux) func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      handler, pattern := mux.Handler(r)
      if handler == nil || pattern == "" {
        WriteError(w, NotFound("not found"))
        return
      }
      next.ServeHTTP(w, r)
    })
  }
}

type traceWriter struct {
  http.ResponseWriter
  statusCode int
  body []byte
}

func (t *traceWriter) WriteHeader(statusCode int) {
  t.statusCode = statusCode
  t.ResponseWriter.WriteHeader(statusCode)
}

func (t *traceWriter) Write(body []byte) (int, error) {
  t.body = body
  return t.ResponseWriter.Write(body)
}

func Trace(reTrace *regexp.Regexp) func(next http.Handler) http.Handler {
  return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      methodPath := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
      if reTrace.MatchString(methodPath) {
        start := time.Now()
        body, _ := io.ReadAll(r.Body)
        r.Body = io.NopCloser(bytes.NewReader(body))
        if len(body) > 0 {
          fmt.Printf("%s %s\n>> %s\n", r.Method, r.URL.Path, udump.JSON(body))
        } else {
          fmt.Printf("%s %s\n", r.Method, r.URL.Path)
        }
        tw := &traceWriter{ResponseWriter: w}
        next.ServeHTTP(tw, r)
        elapsed := time.Since(start).Truncate(time.Millisecond)
        if len(tw.body) > 0 {
          fmt.Printf("<< %d %s %s\n", tw.statusCode, elapsed, udump.JSON(tw.body))
        } else {
          fmt.Printf("<< %d %s\n", tw.statusCode, elapsed)
        }
        return
      }
      next.ServeHTTP(w, r)
    })
  }
}

type logWriter struct {
  http.ResponseWriter
  statusCode int
}

func (l *logWriter) WriteHeader(statusCode int) {
  l.statusCode = statusCode
  l.ResponseWriter.WriteHeader(statusCode)
}

func (l *logWriter) Write(body []byte) (int, error) {
  return l.ResponseWriter.Write(body)
}

type httpLogEntry struct {
  Method string `json:"method"`
  Path string `json:"path"`
  Query string `json:"query,omitempty"`
  StatusCode int `json:"statusCode"`
  Duration int `json:"duration"`
  RemoteIP string `json:"remoteIP"`
  UserAgent string `json:"userAgent"`
  Timestamp time.Time `json:"timestamp"`
}

var reRemoteIP = regexp.MustCompile(`^(\d{1,3}(?:\.\d{1,3}){3}):\d{1,5}`)
var reRemoteIPv6 = regexp.MustCompile(`^\[([a-f\d:.]+)\]`)

func RemoteIP(r *http.Request) string {
  ip := r.RemoteAddr
  header := r.Header.Get("X-Forwarded-For")
  if len(header) > 0 {
    ip = strings.Split(header, ",")[0]
  }
  match := reRemoteIP.FindStringSubmatch(ip)
  if len(match) > 1 {
    ip = match[1]
  }
  match = reRemoteIPv6.FindStringSubmatch(ip)
  if len(match) > 1 {
    ip = match[1]
  }
  return ip
}

func Log(exclude []*regexp.Regexp) func (next http.Handler) http.Handler {
  return func (next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      methodPath := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
      for _, ex := range exclude {
        if ex.MatchString(methodPath) {
          next.ServeHTTP(w, r)
          return
        }
      }
      start := time.Now()
      lw := &logWriter{ResponseWriter: w}
      next.ServeHTTP(lw, r)
      log := httpLogEntry{
        Method: r.Method,
        Path: r.URL.Path,
        Query: r.URL.RawQuery,
        StatusCode: lw.statusCode,
        Duration: int(time.Since(start).Milliseconds()),
        RemoteIP: RemoteIP(r),
        UserAgent: r.UserAgent(),
        Timestamp: time.Now().UTC().Truncate(time.Microsecond),
      }
      jlog, err := json.Marshal(log)
      if err != nil {
        fmt.Fprintf(os.Stderr, "%s\n", err)
        return
      }
      fmt.Printf("%s\n", jlog)
    })
  }
}

type actionLogEntry struct {
  Action string `json:"action"`
  Success bool `json:"success"`
  Error string `json:"error,omitempty"`
  Context []string `json:"context"`
  Duration int `json:"duration"`
  Timestamp time.Time `json:"timestamp"`
}

func LogAction(action string, err error, start time.Time, facts ...string) {
  clean := make([]string, 0, len(facts))
  for _, fact := range facts {
    if len(fact) > 0 {
      clean = append(clean, fact)
    }
  }
  log := actionLogEntry{
    Action: action,
    Success: true,
    Context: clean,
    Duration: int(time.Since(start).Milliseconds()),
    Timestamp: time.Now().UTC().Truncate(time.Microsecond),
  }
  if err != nil {
    log.Success = false
    log.Error = err.Error()
  }
  jlog, err := json.Marshal(log)
  if err != nil {
    fmt.Fprintf(os.Stderr, "%s\n", err)
    return
  }
  fmt.Printf("%s\n", jlog)
}
