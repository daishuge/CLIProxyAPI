package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/interfaces"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/logging"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/logging/conversationlog"
	coreexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"golang.org/x/net/context"
)

const (
	maxConversationBodyCaptureBytes = int64(512 * 1024)
	minConversationBodyCaptureBytes = int64(1024)
)

// SetConversationLogStore updates the opt-in full conversation log sink.
func (h *BaseAPIHandler) SetConversationLogStore(store *conversationlog.Store) {
	if h == nil {
		return
	}
	h.conversationLogMu.Lock()
	defer h.conversationLogMu.Unlock()
	h.conversationLogStore = store
}

func (h *BaseAPIHandler) getConversationLogStore() *conversationlog.Store {
	if h == nil {
		return nil
	}
	h.conversationLogMu.RLock()
	defer h.conversationLogMu.RUnlock()
	return h.conversationLogStore
}

type conversationLogRecorder struct {
	store          *conversationlog.Store
	entry          conversationlog.Entry
	started        time.Time
	responseBudget int64
	responseChunks []string
	responseBytes  int64
	responseCut    bool
	usage          json.RawMessage
}

func (h *BaseAPIHandler) startConversationLog(ctx context.Context, handlerType, operation string, providers []string, modelName string, rawJSON []byte, stream bool, alt string, meta map[string]any) *conversationLogRecorder {
	store := h.getConversationLogStore()
	if store == nil || !store.Enabled() {
		return nil
	}
	started := time.Now().UTC()
	method, path, requestHeaders := conversationRequestInfo(ctx)
	budget := conversationBodyBudget(store)
	entry := conversationlog.Entry{
		RequestID:      logging.GetRequestID(ctx),
		CreatedAt:      started,
		Method:         method,
		Path:           path,
		Provider:       strings.Join(providers, ","),
		Model:          strings.TrimSpace(modelName),
		RequestHeaders: requestHeaders,
		Request:        conversationPayloadFromBytes(rawJSON, budget),
		Metadata:       conversationMetadata(handlerType, operation, stream, alt, meta),
	}
	return &conversationLogRecorder{
		store:          store,
		entry:          entry,
		started:        started,
		responseBudget: budget,
	}
}

// beginConversationLog resolves the request-time metadata recorded by the
// conversation log and starts a recorder when logging is enabled. It returns the
// (possibly augmented) context, the recorder (nil when logging is disabled, in
// which case the caller skips every recorder call), and the shared metadata map
// that is merged into the entry when the request finishes.
//
// The selected auth ID is published asynchronously by the auth scheduler, so the
// returned context carries a callback that writes the chosen auth ID back into
// reqMeta. The map is only read at finish time, after execution has returned, so
// there is no concurrent read/write race with the scheduler write.
func (h *BaseAPIHandler) beginConversationLog(ctx context.Context, handlerType, operation, modelName string, rawJSON []byte, stream bool, alt string) (context.Context, *conversationLogRecorder, map[string]any) {
	store := h.getConversationLogStore()
	if store == nil || !store.Enabled() {
		return ctx, nil, nil
	}
	// Best-effort provider resolution for the recorded entry. When the model is
	// unknown the inner execute call returns the same error and the recorder still
	// captures an error entry with an empty provider list.
	providers, normalizedModel, errMsg := h.getRequestDetails(modelName)
	if errMsg != nil {
		providers = nil
		normalizedModel = modelName
	}
	reqMeta := requestExecutionMetadata(ctx)
	reqMeta[coreexecutor.RequestedModelMetadataKey] = normalizedModel
	ctx = WithSelectedAuthIDCallback(ctx, func(authID string) {
		authID = strings.TrimSpace(authID)
		if authID == "" {
			return
		}
		reqMeta[coreexecutor.SelectedAuthMetadataKey] = authID
	})
	rec := h.startConversationLog(ctx, handlerType, operation, providers, normalizedModel, rawJSON, stream, alt, reqMeta)
	return ctx, rec, reqMeta
}

// conversationResponseHeaders resolves the response headers recorded for a
// conversation log entry. The downstream proxy returns nil headers when passthrough
// headers are disabled (the default), so this falls back to the raw upstream
// headers published into the context holder by the execute path.
func conversationResponseHeaders(ctx context.Context, headers http.Header) http.Header {
	if len(headers) > 0 {
		return headers
	}
	if raw := logging.GetResponseHeaders(ctx); len(raw) > 0 {
		return raw
	}
	return headers
}

// teeConversationLogStream forwards the upstream streaming channels to the client
// while capturing each chunk and the terminal error into the recorder. The
// conversation log entry is written once both upstream channels close.
//
// clientHeaders are returned unchanged to the caller (nil when passthrough headers
// are disabled). loggedHeaders are recorded in the conversation log entry and carry
// the raw upstream stream headers even when the client sees none.
//
// Forwarding sends are guarded by ctx so that a client disconnect (cancelled ctx)
// cannot deadlock the tee goroutine: the upstream stream goroutine already honors
// ctx.Done(), and matching that here keeps the recorder from leaking goroutines.
func teeConversationLogStream(ctx context.Context, rec *conversationLogRecorder, clientHeaders, loggedHeaders http.Header, reqMeta map[string]any, upstreamData <-chan []byte, upstreamErrs <-chan *interfaces.ErrorMessage) (<-chan []byte, http.Header, <-chan *interfaces.ErrorMessage) {
	dataChan := make(chan []byte)
	errChan := make(chan *interfaces.ErrorMessage, 1)
	var done <-chan struct{}
	if ctx != nil {
		done = ctx.Done()
	}
	go func() {
		status := http.StatusOK
		var finalErr error
		var addon http.Header
		dataOpen := upstreamData != nil
		errsOpen := upstreamErrs != nil
		for dataOpen || errsOpen {
			select {
			case chunk, ok := <-upstreamData:
				if !ok {
					upstreamData = nil
					dataOpen = false
					continue
				}
				rec.captureStreamChunk(chunk)
				select {
				case dataChan <- chunk:
				case <-done:
					dataOpen = false
					upstreamData = nil
				}
			case msg, ok := <-upstreamErrs:
				if !ok {
					upstreamErrs = nil
					errsOpen = false
					continue
				}
				if msg != nil {
					if msg.StatusCode > 0 {
						status = msg.StatusCode
					} else {
						status = http.StatusInternalServerError
					}
					finalErr = msg.Error
					addon = msg.Addon
				}
				select {
				case errChan <- msg:
				case <-done:
				}
				errsOpen = false
				upstreamErrs = nil
			}
		}
		responseHeaders := loggedHeaders
		if addon != nil {
			responseHeaders = addon
		}
		rec.finishStream(responseHeaders, status, finalErr, reqMeta)
		close(dataChan)
		close(errChan)
	}()
	return dataChan, clientHeaders, errChan
}

func (h *BaseAPIHandler) finishConversationLogError(ctx context.Context, handlerType, operation, modelName string, rawJSON []byte, stream bool, alt string, meta map[string]any, msg *interfaces.ErrorMessage) {
	rec := h.startConversationLog(ctx, handlerType, operation, nil, modelName, rawJSON, stream, alt, meta)
	if rec == nil {
		return
	}
	status := http.StatusInternalServerError
	var err error
	var headers http.Header
	if msg != nil {
		if msg.StatusCode > 0 {
			status = msg.StatusCode
		}
		err = msg.Error
		headers = msg.Addon
	}
	if stream {
		rec.finishStream(headers, status, err, meta)
		return
	}
	rec.finishNonStream(nil, headers, status, err, meta)
}

func (r *conversationLogRecorder) captureStreamChunk(chunk []byte) {
	if r == nil || len(chunk) == 0 {
		return
	}
	r.responseBytes += int64(len(chunk))
	if usage := extractConversationUsage(chunk); len(usage) > 0 {
		r.usage = usage
	}
	if r.responseBudget <= 0 {
		r.responseCut = true
		return
	}
	redacted := redactConversationPayloadBytes(chunk)
	remaining := r.responseBudget
	for _, existing := range r.responseChunks {
		remaining -= int64(len(existing))
	}
	if remaining <= 0 {
		r.responseCut = true
		return
	}
	if int64(len(redacted)) > remaining {
		redacted = truncateConversationBytes(redacted, remaining)
		r.responseCut = true
	}
	r.responseChunks = append(r.responseChunks, string(redacted))
}

func (r *conversationLogRecorder) finishNonStream(payload []byte, headers http.Header, status int, err error, meta map[string]any) {
	if r == nil {
		return
	}
	r.entry.ResponseHeaders = headerMap(headers)
	r.entry.Response = conversationPayloadFromBytes(payload, r.responseBudget)
	r.entry.Usage = extractConversationUsage(payload)
	r.finish(status, err, meta)
}

func (r *conversationLogRecorder) finishStream(headers http.Header, status int, err error, meta map[string]any) {
	if r == nil {
		return
	}
	r.entry.ResponseHeaders = headerMap(headers)
	r.entry.Response = conversationlog.Payload{
		Chunks:    r.responseChunks,
		Bytes:     r.responseBytes,
		Truncated: r.responseCut,
	}
	r.entry.Usage = r.usage
	r.finish(status, err, meta)
}

func (r *conversationLogRecorder) finish(status int, err error, meta map[string]any) {
	r.entry.CompletedAt = time.Now().UTC()
	r.entry.LatencyMS = r.entry.CompletedAt.Sub(r.started).Milliseconds()
	if status > 0 {
		r.entry.StatusCode = status
	}
	if err != nil {
		r.entry.Error = truncateConversationString(err.Error(), 4096)
	}
	r.entry.Metadata = mergeConversationMetadata(r.entry.Metadata, conversationMetadata("", "", false, "", meta))
	if _, writeErr := r.store.Write(r.entry); writeErr != nil {
		log.WithError(writeErr).Warn("failed to write conversation log entry")
	}
}

func conversationRequestInfo(ctx context.Context) (string, string, map[string][]string) {
	ginCtx := ginContextFromContext(ctx)
	if ginCtx != nil && ginCtx.Request != nil {
		method := strings.TrimSpace(ginCtx.Request.Method)
		path := strings.TrimSpace(ginCtx.FullPath())
		if path == "" && ginCtx.Request.URL != nil {
			path = strings.TrimSpace(ginCtx.Request.URL.Path)
		}
		return method, path, headerMap(ginCtx.Request.Header)
	}
	endpoint := strings.TrimSpace(logging.GetEndpoint(ctx))
	if endpoint != "" {
		parts := strings.SplitN(endpoint, " ", 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
		}
		return "", endpoint, nil
	}
	return "", "", nil
}

func ginContextFromContext(ctx context.Context) *gin.Context {
	if ctx == nil {
		return nil
	}
	ginCtx, _ := ctx.Value("gin").(*gin.Context)
	return ginCtx
}

func headerMap(headers http.Header) map[string][]string {
	if len(headers) == 0 {
		return nil
	}
	return headers.Clone()
}

func conversationMetadata(handlerType, operation string, stream bool, alt string, meta map[string]any) map[string]string {
	out := map[string]string{}
	if handlerType = strings.TrimSpace(handlerType); handlerType != "" {
		out["handler_type"] = handlerType
	}
	if operation = strings.TrimSpace(operation); operation != "" {
		out["operation"] = operation
	}
	if alt = strings.TrimSpace(alt); alt != "" {
		out["alt"] = alt
	}
	if stream {
		out["stream"] = strconv.FormatBool(stream)
	}
	for _, key := range []string{
		coreexecutor.RequestedModelMetadataKey,
		coreexecutor.RequestPathMetadataKey,
		coreexecutor.PinnedAuthMetadataKey,
		coreexecutor.SelectedAuthMetadataKey,
		coreexecutor.ExecutionSessionMetadataKey,
		coreexecutor.DisallowFreeAuthMetadataKey,
		idempotencyKeyMetadataKey,
	} {
		if value := conversationMetadataValue(meta[key]); value != "" {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func mergeConversationMetadata(base, extra map[string]string) map[string]string {
	if len(base) == 0 {
		if len(extra) == 0 {
			return nil
		}
		out := make(map[string]string, len(extra))
		for key, value := range extra {
			out[key] = value
		}
		return out
	}
	out := make(map[string]string, len(base)+len(extra))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range extra {
		out[key] = value
	}
	return out
}

func conversationMetadataValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []byte:
		return strings.TrimSpace(string(typed))
	case bool:
		return strconv.FormatBool(typed)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return ""
	}
}

func conversationBodyBudget(store *conversationlog.Store) int64 {
	if store == nil {
		return maxConversationBodyCaptureBytes
	}
	budget := store.MaxEntryBytes() / 4
	if budget <= 0 {
		return maxConversationBodyCaptureBytes
	}
	if budget > maxConversationBodyCaptureBytes {
		budget = maxConversationBodyCaptureBytes
	}
	if budget < minConversationBodyCaptureBytes {
		budget = minConversationBodyCaptureBytes
	}
	return budget
}

func conversationPayloadFromBytes(raw []byte, budget int64) conversationlog.Payload {
	if len(raw) == 0 {
		return conversationlog.Payload{}
	}
	total := int64(len(raw))
	rendered := redactConversationPayloadBytes(raw)
	trimmed := bytes.TrimSpace(rendered)
	jsonPayload := json.Valid(trimmed)
	if jsonPayload {
		rendered = trimmed
	}
	truncated := false
	if budget > 0 && int64(len(rendered)) > budget {
		rendered = truncateConversationBytes(rendered, budget)
		truncated = true
	}
	payload := conversationlog.Payload{
		Bytes:     total,
		Truncated: truncated,
	}
	if jsonPayload && json.Valid(rendered) {
		payload.Body = json.RawMessage(bytes.Clone(rendered))
		return payload
	}
	payload.Text = string(rendered)
	return payload
}

func redactConversationPayloadBytes(raw []byte) []byte {
	if len(raw) == 0 {
		return nil
	}
	trimmed := bytes.TrimSpace(raw)
	if json.Valid(trimmed) {
		return []byte(conversationlog.RedactJSON(json.RawMessage(trimmed)))
	}
	if bytes.Contains(raw, []byte("data:")) {
		lines := bytes.Split(raw, []byte("\n"))
		changed := false
		for i, line := range lines {
			prefixLen := sseDataPrefixLength(line)
			if prefixLen < 0 {
				continue
			}
			data := bytes.TrimSpace(line[prefixLen:])
			if len(data) == 0 || bytes.Equal(data, []byte("[DONE]")) || !json.Valid(data) {
				continue
			}
			prefix := bytes.Clone(line[:prefixLen])
			lines[i] = append(prefix, conversationlog.RedactJSON(json.RawMessage(data))...)
			changed = true
		}
		if changed {
			return bytes.Join(lines, []byte("\n"))
		}
	}
	return bytes.Clone(raw)
}

func sseDataPrefixLength(line []byte) int {
	trimmed := bytes.TrimLeft(line, " \t")
	offset := len(line) - len(trimmed)
	if !bytes.HasPrefix(trimmed, []byte("data:")) {
		return -1
	}
	return offset + len("data:")
}

func extractConversationUsage(raw []byte) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	if usage := extractUsageFromJSON(bytes.TrimSpace(raw)); len(usage) > 0 {
		return usage
	}
	for _, line := range bytes.Split(raw, []byte("\n")) {
		prefixLen := sseDataPrefixLength(line)
		if prefixLen < 0 {
			continue
		}
		data := bytes.TrimSpace(line[prefixLen:])
		if len(data) == 0 || bytes.Equal(data, []byte("[DONE]")) {
			continue
		}
		if usage := extractUsageFromJSON(data); len(usage) > 0 {
			return usage
		}
	}
	return nil
}

func extractUsageFromJSON(raw []byte) json.RawMessage {
	if !json.Valid(raw) {
		return nil
	}
	for _, path := range []string{"usage", "response.usage"} {
		result := gjson.GetBytes(raw, path)
		if !result.Exists() || result.Raw == "" || result.Type == gjson.Null {
			continue
		}
		candidate := []byte(result.Raw)
		if json.Valid(candidate) {
			return json.RawMessage(bytes.Clone(candidate))
		}
	}
	return nil
}

func truncateConversationString(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	return string(truncateConversationBytes([]byte(value), int64(max))) + "...[truncated]"
}

func truncateConversationBytes(value []byte, max int64) []byte {
	if max <= 0 {
		return nil
	}
	if int64(len(value)) <= max {
		return value
	}
	cut := int(max)
	out := value[:cut]
	if !utf8.Valid(value) {
		return out
	}
	for len(out) > 0 && !utf8.Valid(out) {
		out = out[:len(out)-1]
	}
	return out
}
