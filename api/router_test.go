package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggingMiddlewareLogsNormalResponse(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, nil))

	router := mux.NewRouter()
	router.HandleFunc("/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.Header.Get("X-Request-ID"))
		_, err := io.WriteString(w, "hello")
		require.NoError(t, err)
	}).Methods(http.MethodGet)
	router.Use(func(next http.Handler) http.Handler {
		return loggingMiddlewareWithLogger(next, logger)
	})

	req := httptest.NewRequest(http.MethodGet, "/items/42", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	req.Header.Set("User-Agent", "nahcloud-test/1.0")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "hello", response.Body.String())
	requestID := response.Header().Get("X-Request-ID")
	require.NoError(t, uuid.Validate(requestID))

	entry := decodeLogEntry(t, &logOutput)
	assert.Equal(t, "http_request", entry["msg"])
	assert.Equal(t, http.MethodGet, entry["method"])
	assert.Equal(t, "/items/{id}", entry["route"])
	assert.Equal(t, "/items/42", entry["path"])
	assert.Equal(t, float64(http.StatusOK), entry["status"])
	assert.Equal(t, float64(len("hello")), entry["response_bytes"])
	assert.NotEmpty(t, entry["duration"])
	assert.Equal(t, "192.0.2.10:1234", entry["remote_addr"])
	assert.Equal(t, "nahcloud-test/1.0", entry["user_agent"])
	assert.Equal(t, requestID, entry["request_id"])
}

func TestLoggingMiddlewareRecordsCustomStatusAndResponseBytes(t *testing.T) {
	var logOutput bytes.Buffer
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, err := io.WriteString(w, "created")
		require.NoError(t, err)
	})

	req := httptest.NewRequest(http.MethodPost, "/items", nil)
	response := httptest.NewRecorder()
	loggingMiddlewareWithLogger(next, slog.New(slog.NewJSONHandler(&logOutput, nil))).ServeHTTP(response, req)

	require.Equal(t, http.StatusCreated, response.Code)
	entry := decodeLogEntry(t, &logOutput)
	assert.Equal(t, float64(http.StatusCreated), entry["status"])
	assert.Equal(t, float64(len("created")), entry["response_bytes"])
}

func TestLoggingMiddlewareRequestID(t *testing.T) {
	tests := []struct {
		name       string
		incomingID string
	}{
		{name: "propagates incoming ID", incomingID: "upstream-request-123"},
		{name: "generates ID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logOutput bytes.Buffer
			var handlerRequestID string
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerRequestID = r.Header.Get("X-Request-ID")
				w.WriteHeader(http.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			if tt.incomingID != "" {
				req.Header.Set("X-Request-ID", tt.incomingID)
			}
			response := httptest.NewRecorder()
			loggingMiddlewareWithLogger(next, slog.New(slog.NewJSONHandler(&logOutput, nil))).ServeHTTP(response, req)

			requestID := response.Header().Get("X-Request-ID")
			if tt.incomingID != "" {
				assert.Equal(t, tt.incomingID, requestID)
			} else {
				require.NoError(t, uuid.Validate(requestID))
			}
			assert.Equal(t, requestID, handlerRequestID)
			assert.Equal(t, requestID, decodeLogEntry(t, &logOutput)["request_id"])
		})
	}
}

func TestResponseRecorderPreservesWriterBehavior(t *testing.T) {
	var logOutput bytes.Buffer
	underlying := newOptionalResponseWriter()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)
		flusher.Flush()

		pusher, ok := w.(http.Pusher)
		require.True(t, ok)
		require.NoError(t, pusher.Push("/static/app.css", nil))

		hijacker, ok := w.(http.Hijacker)
		require.True(t, ok)
		_, _, err := hijacker.Hijack()
		require.ErrorIs(t, err, http.ErrNotSupported)

		readerFrom, ok := w.(io.ReaderFrom)
		require.True(t, ok)
		n, err := readerFrom.ReadFrom(strings.NewReader("streamed"))
		require.NoError(t, err)
		assert.Equal(t, int64(len("streamed")), n)
	})

	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	loggingMiddlewareWithLogger(next, slog.New(slog.NewJSONHandler(&logOutput, nil))).ServeHTTP(underlying, req)

	assert.True(t, underlying.flushed)
	assert.Equal(t, "/static/app.css", underlying.pushed)
	assert.Equal(t, "streamed", underlying.body.String())
	assert.Equal(t, float64(len("streamed")), decodeLogEntry(t, &logOutput)["response_bytes"])
}

func TestResponseRecorderDoesNotAdvertiseUnsupportedInterfaces(t *testing.T) {
	var logOutput bytes.Buffer
	underlying := newBasicResponseWriter()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, flushes := w.(http.Flusher)
		_, hijacks := w.(http.Hijacker)
		_, pushes := w.(http.Pusher)
		_, readsFrom := w.(io.ReaderFrom)
		assert.False(t, flushes)
		assert.False(t, hijacks)
		assert.False(t, pushes)
		assert.False(t, readsFrom)
	})

	req := httptest.NewRequest(http.MethodGet, "/basic", nil)
	loggingMiddlewareWithLogger(next, slog.New(slog.NewJSONHandler(&logOutput, nil))).ServeHTTP(underlying, req)
}

func TestSetupRouterLogsNotFoundResponse(t *testing.T) {
	var logOutput bytes.Buffer
	previousLogger := requestLogger
	requestLogger = slog.New(slog.NewJSONHandler(&logOutput, nil))
	t.Cleanup(func() { requestLogger = previousLogger })
	router := SetupRouter(NewHandler(nil), nil, "test")

	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	assert.Equal(t, http.StatusNotFound, response.Code)
	require.NoError(t, uuid.Validate(response.Header().Get("X-Request-ID")))
	entry := decodeLogEntry(t, &logOutput)
	assert.Equal(t, "/does-not-exist", entry["route"])
	assert.Equal(t, float64(http.StatusNotFound), entry["status"])
}

func decodeLogEntry(t *testing.T, output *bytes.Buffer) map[string]any {
	t.Helper()
	var entry map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &entry))
	return entry
}

type optionalResponseWriter struct {
	header  http.Header
	body    bytes.Buffer
	status  int
	flushed bool
	pushed  string
}

func newOptionalResponseWriter() *optionalResponseWriter {
	return &optionalResponseWriter{header: make(http.Header)}
}

func (w *optionalResponseWriter) Header() http.Header {
	return w.header
}

func (w *optionalResponseWriter) Write(p []byte) (int, error) {
	return w.body.Write(p)
}

func (w *optionalResponseWriter) WriteHeader(status int) {
	w.status = status
}

func (w *optionalResponseWriter) Flush() {
	w.flushed = true
}

func (w *optionalResponseWriter) Push(target string, _ *http.PushOptions) error {
	w.pushed = target
	return nil
}

func (w *optionalResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, http.ErrNotSupported
}

func (w *optionalResponseWriter) ReadFrom(r io.Reader) (int64, error) {
	return w.body.ReadFrom(r)
}

type basicResponseWriter struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func newBasicResponseWriter() *basicResponseWriter {
	return &basicResponseWriter{header: make(http.Header)}
}

func (w *basicResponseWriter) Header() http.Header {
	return w.header
}

func (w *basicResponseWriter) Write(p []byte) (int, error) {
	return w.body.Write(p)
}

func (w *basicResponseWriter) WriteHeader(status int) {
	w.status = status
}
