// Package conversationlog stores opt-in full conversation logs as bounded JSONL shards.
package conversationlog

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	appconfig "github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

const (
	filePrefix       = "conversation-"
	fileSuffix       = ".jsonl"
	defaultListLimit = 100
	maxListLimit     = 500
)

var (
	ErrDisabled      = errors.New("conversation log disabled")
	ErrEntryTooLarge = errors.New("conversation log entry too large")
	ErrNotFound      = errors.New("conversation log entry not found")
)

// Options configures conversation log storage.
type Options struct {
	Enabled           bool
	Directory         string
	MaxFileSizeBytes  int64
	MaxTotalSizeBytes int64
	MaxEntryBytes     int64
}

// OptionsFromConfig converts application config into storage options.
func OptionsFromConfig(cfg *appconfig.Config, configPath string) Options {
	logCfg := appconfig.DefaultConversationLogConfig()
	if cfg != nil {
		logCfg = cfg.ConversationLog
		logCfg.Normalize()
	}

	dir := logCfg.Directory
	if dir == "" {
		dir = appconfig.DefaultConversationLogDir
	}
	if !filepath.IsAbs(dir) && strings.TrimSpace(configPath) != "" {
		dir = filepath.Join(filepath.Dir(configPath), dir)
	}

	return normalizeOptions(Options{
		Enabled:           logCfg.Enabled,
		Directory:         dir,
		MaxFileSizeBytes:  megabytesToBytes(logCfg.MaxFileSizeMB),
		MaxTotalSizeBytes: megabytesToBytes(logCfg.MaxTotalSizeMB),
		MaxEntryBytes:     int64(logCfg.MaxEntryBytes),
	})
}

func normalizeOptions(opts Options) Options {
	defaults := appconfig.DefaultConversationLogConfig()
	if strings.TrimSpace(opts.Directory) == "" {
		opts.Directory = defaults.Directory
	}
	opts.Directory = filepath.Clean(strings.TrimSpace(opts.Directory))
	if opts.MaxFileSizeBytes <= 0 {
		opts.MaxFileSizeBytes = megabytesToBytes(defaults.MaxFileSizeMB)
	}
	if opts.MaxTotalSizeBytes <= 0 {
		opts.MaxTotalSizeBytes = megabytesToBytes(defaults.MaxTotalSizeMB)
	}
	if opts.MaxEntryBytes <= 0 {
		opts.MaxEntryBytes = int64(defaults.MaxEntryBytes)
	}
	return opts
}

func megabytesToBytes(value int) int64 {
	if value <= 0 {
		return 0
	}
	return int64(value) * 1024 * 1024
}

// Store writes and reads full conversation log entries.
type Store struct {
	opts Options
	now  func() time.Time
	mu   sync.Mutex
}

// NewStore returns a file-backed conversation log store.
func NewStore(opts Options) *Store {
	return &Store{
		opts: normalizeOptions(opts),
		now:  time.Now,
	}
}

// Enabled reports whether this store will persist entries.
func (s *Store) Enabled() bool {
	return s != nil && s.opts.Enabled
}

// MaxEntryBytes returns the normalized per-entry byte budget.
func (s *Store) MaxEntryBytes() int64 {
	if s == nil {
		return int64(appconfig.DefaultConversationLogEntryBytes)
	}
	return s.opts.MaxEntryBytes
}

// SetNowForTest overrides the clock used by Store. It is intended for tests.
func (s *Store) SetNowForTest(now func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if now == nil {
		s.now = time.Now
		return
	}
	s.now = now
}

// Entry is one full conversation log record.
type Entry struct {
	ID              string              `json:"id"`
	RequestID       string              `json:"request_id,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	CompletedAt     time.Time           `json:"completed_at,omitempty"`
	LatencyMS       int64               `json:"latency_ms,omitempty"`
	Method          string              `json:"method,omitempty"`
	Path            string              `json:"path,omitempty"`
	Provider        string              `json:"provider,omitempty"`
	Model           string              `json:"model,omitempty"`
	UpstreamURL     string              `json:"upstream_url,omitempty"`
	StatusCode      int                 `json:"status_code,omitempty"`
	Error           string              `json:"error,omitempty"`
	RequestHeaders  map[string][]string `json:"request_headers,omitempty"`
	ResponseHeaders map[string][]string `json:"response_headers,omitempty"`
	Request         Payload             `json:"request,omitempty"`
	Response        Payload             `json:"response,omitempty"`
	Usage           json.RawMessage     `json:"usage,omitempty"`
	Metadata        map[string]string   `json:"metadata,omitempty"`
}

// Payload stores a request or response body snapshot.
type Payload struct {
	Body      json.RawMessage `json:"body,omitempty"`
	Text      string          `json:"text,omitempty"`
	Chunks    []string        `json:"chunks,omitempty"`
	Bytes     int64           `json:"bytes,omitempty"`
	Truncated bool            `json:"truncated,omitempty"`
}

// Location describes where an entry was written.
type Location struct {
	ID   string `json:"id"`
	File string `json:"file"`
	Size int64  `json:"size"`
}

// EntrySummary is the lightweight shape used for log lists.
type EntrySummary struct {
	ID         string    `json:"id"`
	RequestID  string    `json:"request_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	Method     string    `json:"method,omitempty"`
	Path       string    `json:"path,omitempty"`
	Provider   string    `json:"provider,omitempty"`
	Model      string    `json:"model,omitempty"`
	StatusCode int       `json:"status_code,omitempty"`
	HasError   bool      `json:"has_error"`
	File       string    `json:"file"`
	LineBytes  int64     `json:"line_bytes"`
}

// ListQuery controls list pagination.
type ListQuery struct {
	Limit      int
	Cursor     string
	RequestID  string
	Provider   string
	Model      string
	Path       string
	StatusCode *int
	HasError   *bool
	From       time.Time
	To         time.Time
}

// ListResult contains paginated summaries.
type ListResult struct {
	Entries    []EntrySummary `json:"entries"`
	NextCursor string         `json:"next_cursor,omitempty"`
	Malformed  int            `json:"malformed"`
}

// Write persists an entry when the store is enabled.
func (s *Store) Write(entry Entry) (Location, error) {
	if s == nil {
		return Location{}, ErrDisabled
	}
	if !s.opts.Enabled {
		return Location{}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if err := entry.prepare(now); err != nil {
		return Location{}, err
	}

	payload, err := json.Marshal(entry)
	if err != nil {
		return Location{}, fmt.Errorf("marshal conversation log entry: %w", err)
	}
	if int64(len(payload)) > s.opts.MaxEntryBytes {
		return Location{}, ErrEntryTooLarge
	}
	line := append(payload, '\n')

	if err := os.MkdirAll(s.opts.Directory, 0o700); err != nil {
		return Location{}, fmt.Errorf("create conversation log directory: %w", err)
	}
	_ = os.Chmod(s.opts.Directory, 0o700)

	path, err := s.currentFilePath(int64(len(line)), now)
	if err != nil {
		return Location{}, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return Location{}, fmt.Errorf("open conversation log file: %w", err)
	}
	_ = file.Chmod(0o600)
	if _, errWrite := file.Write(line); errWrite != nil {
		_ = file.Close()
		return Location{}, fmt.Errorf("write conversation log entry: %w", errWrite)
	}
	if errClose := file.Close(); errClose != nil {
		return Location{}, fmt.Errorf("close conversation log file: %w", errClose)
	}

	if errRetention := s.enforceRetention(path); errRetention != nil {
		return Location{}, errRetention
	}

	return Location{ID: entry.ID, File: path, Size: int64(len(line))}, nil
}

func (entry *Entry) prepare(now time.Time) error {
	if entry.ID == "" {
		entry.ID = generateID()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now.UTC()
	}
	if len(entry.Request.Body) > 0 && !json.Valid(entry.Request.Body) {
		return fmt.Errorf("request body is not valid JSON")
	}
	if len(entry.Response.Body) > 0 && !json.Valid(entry.Response.Body) {
		return fmt.Errorf("response body is not valid JSON")
	}
	if len(entry.Usage) > 0 && !json.Valid(entry.Usage) {
		return fmt.Errorf("usage body is not valid JSON")
	}
	entry.Method = strings.TrimSpace(entry.Method)
	entry.Path = strings.TrimSpace(entry.Path)
	entry.Provider = strings.TrimSpace(entry.Provider)
	entry.Model = strings.TrimSpace(entry.Model)
	entry.UpstreamURL = RedactURL(entry.UpstreamURL)
	entry.RequestHeaders = RedactHeaders(entry.RequestHeaders)
	entry.ResponseHeaders = RedactHeaders(entry.ResponseHeaders)
	entry.Metadata = RedactMetadata(entry.Metadata)
	return nil
}

func (s *Store) currentFilePath(lineSize int64, now time.Time) (string, error) {
	if lineSize > s.opts.MaxFileSizeBytes {
		return "", ErrEntryTooLarge
	}

	files, err := listConversationFiles(s.opts.Directory, false)
	if err != nil {
		return "", err
	}
	if len(files) > 0 {
		current := files[len(files)-1]
		if info, errStat := os.Stat(current); errStat == nil && info.Size()+lineSize <= s.opts.MaxFileSizeBytes {
			return current, nil
		}
	}
	return filepath.Join(s.opts.Directory, newFileName(now)), nil
}

func newFileName(now time.Time) string {
	return filePrefix + now.UTC().Format("20060102T150405.000000000Z") + "-" + generateID() + fileSuffix
}

func generateID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b[:])
}

func (s *Store) enforceRetention(protectedPath string) error {
	if s.opts.MaxTotalSizeBytes <= 0 {
		return nil
	}
	files, err := listConversationFiles(s.opts.Directory, false)
	if err != nil {
		return err
	}
	type fileInfo struct {
		path    string
		size    int64
		modTime time.Time
	}
	items := make([]fileInfo, 0, len(files))
	var total int64
	for _, path := range files {
		info, errInfo := os.Stat(path)
		if errInfo != nil || !info.Mode().IsRegular() {
			continue
		}
		items = append(items, fileInfo{path: path, size: info.Size(), modTime: info.ModTime()})
		total += info.Size()
	}
	if total <= s.opts.MaxTotalSizeBytes {
		return nil
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].modTime.Equal(items[j].modTime) {
			return items[i].path < items[j].path
		}
		return items[i].modTime.Before(items[j].modTime)
	})
	protected := filepath.Clean(protectedPath)
	for _, item := range items {
		if total <= s.opts.MaxTotalSizeBytes {
			return nil
		}
		if filepath.Clean(item.path) == protected {
			continue
		}
		if errRemove := os.Remove(item.path); errRemove != nil && !os.IsNotExist(errRemove) {
			return fmt.Errorf("remove old conversation log file: %w", errRemove)
		}
		total -= item.size
	}
	return nil
}

// List returns entry summaries newest first.
func (s *Store) List(query ListQuery) (ListResult, error) {
	result := ListResult{Entries: []EntrySummary{}}
	if s == nil || !s.opts.Enabled {
		return result, nil
	}
	limit := normalizeLimit(query.Limit)
	skip, err := parseCursor(query.Cursor)
	if err != nil {
		return result, err
	}

	files, err := listConversationFiles(s.opts.Directory, true)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return result, err
	}

	seen := 0
	stoppedEarly := false
	for _, path := range files {
		summaries, malformed, errRead := readSummaries(path, s.opts.MaxEntryBytes)
		result.Malformed += malformed
		if errRead != nil {
			return result, errRead
		}
		for i := len(summaries) - 1; i >= 0; i-- {
			if seen < skip {
				seen++
				continue
			}
			summary := summaries[i]
			if !summaryMatchesListQuery(summary, query) {
				seen++
				continue
			}
			if len(result.Entries) >= limit {
				stoppedEarly = true
				break
			}
			result.Entries = append(result.Entries, summary)
			seen++
		}
		if stoppedEarly {
			break
		}
	}
	if stoppedEarly {
		result.NextCursor = strconv.Itoa(seen)
	}
	return result, nil
}

// Read finds a single entry by ID.
func (s *Store) Read(id string) (Entry, error) {
	if s == nil || !s.opts.Enabled {
		return Entry{}, ErrNotFound
	}
	id = strings.TrimSpace(id)
	if id == "" || strings.ContainsAny(id, `/\`) {
		return Entry{}, ErrNotFound
	}

	files, err := listConversationFiles(s.opts.Directory, true)
	if err != nil {
		if os.IsNotExist(err) {
			return Entry{}, ErrNotFound
		}
		return Entry{}, err
	}
	for _, path := range files {
		entry, found, errRead := readEntry(path, id, s.opts.MaxEntryBytes)
		if errRead != nil {
			return Entry{}, errRead
		}
		if found {
			return entry, nil
		}
	}
	return Entry{}, ErrNotFound
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultListLimit
	}
	if limit > maxListLimit {
		return maxListLimit
	}
	return limit
}

func parseCursor(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	cursor, err := strconv.Atoi(value)
	if err != nil || cursor < 0 {
		return 0, fmt.Errorf("invalid conversation log cursor")
	}
	return cursor, nil
}

func summaryMatchesListQuery(summary EntrySummary, query ListQuery) bool {
	if !containsFold(summary.RequestID, query.RequestID) {
		return false
	}
	if !containsFold(summary.Provider, query.Provider) {
		return false
	}
	if !containsFold(summary.Model, query.Model) {
		return false
	}
	if !containsFold(summary.Path, query.Path) {
		return false
	}
	if query.StatusCode != nil && summary.StatusCode != *query.StatusCode {
		return false
	}
	if query.HasError != nil && summary.HasError != *query.HasError {
		return false
	}
	if !query.From.IsZero() && summary.CreatedAt.Before(query.From) {
		return false
	}
	if !query.To.IsZero() && summary.CreatedAt.After(query.To) {
		return false
	}
	return true
}

func containsFold(value string, filter string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}
	return strings.Contains(strings.ToLower(value), strings.ToLower(filter))
}

func listConversationFiles(dir string, newestFirst bool) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !isConversationFileName(name) {
			continue
		}
		files = append(files, filepath.Join(dir, name))
	}
	sort.Strings(files)
	if newestFirst {
		for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
			files[i], files[j] = files[j], files[i]
		}
	}
	return files, nil
}

func isConversationFileName(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	return strings.HasPrefix(lower, filePrefix) && strings.HasSuffix(lower, fileSuffix)
}

func readSummaries(path string, maxEntryBytes int64) ([]EntrySummary, int, error) {
	summaries := []EntrySummary{}
	malformedJSON := 0
	malformedOversized, err := readLogLines(path, maxEntryBytes, func(line []byte) error {
		var entry Entry
		if errJSON := json.Unmarshal(line, &entry); errJSON != nil {
			malformedJSON++
			return nil
		}
		summaries = append(summaries, summarizeEntry(entry, filepath.Base(path), int64(len(line))))
		return nil
	})
	if err != nil {
		return summaries, malformedJSON + malformedOversized, err
	}
	return summaries, malformedJSON + malformedOversized, nil
}

func readEntry(path string, id string, maxEntryBytes int64) (Entry, bool, error) {
	var found Entry
	foundEntry := false
	errStop := errors.New("stop reading conversation log")
	_, err := readLogLines(path, maxEntryBytes, func(line []byte) error {
		var entry Entry
		if errJSON := json.Unmarshal(line, &entry); errJSON != nil {
			return nil
		}
		if entry.ID == id {
			found = entry
			foundEntry = true
			return errStop
		}
		return nil
	})
	if errors.Is(err, errStop) {
		return found, true, nil
	}
	if err != nil {
		return Entry{}, false, err
	}
	return found, foundEntry, nil
}

func readLogLines(path string, maxEntryBytes int64, visit func([]byte) error) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = file.Close()
	}()

	reader := bufio.NewReaderSize(file, 64*1024)
	maxLineBytes := scannerMaxBuffer(maxEntryBytes)
	line := make([]byte, 0, 64*1024)
	malformed := 0
	oversized := false
	for {
		part, errRead := reader.ReadSlice('\n')
		if len(part) > 0 && !oversized {
			if len(line)+len(part) > maxLineBytes {
				oversized = true
				line = line[:0]
			} else {
				line = append(line, part...)
			}
		}
		if errors.Is(errRead, bufio.ErrBufferFull) {
			continue
		}
		if errRead != nil && !errors.Is(errRead, io.EOF) {
			return malformed, errRead
		}
		if oversized {
			malformed++
			oversized = false
			line = line[:0]
		} else {
			line = bytes.TrimRight(line, "\r\n")
			if len(strings.TrimSpace(string(line))) > 0 {
				if errVisit := visit(line); errVisit != nil {
					return malformed, errVisit
				}
			}
			line = line[:0]
		}
		if errors.Is(errRead, io.EOF) {
			break
		}
	}
	return malformed, nil
}

func scannerMaxBuffer(maxEntryBytes int64) int {
	if maxEntryBytes <= 0 {
		return appconfig.DefaultConversationLogEntryBytes + 64*1024
	}
	if maxEntryBytes > int64(^uint(0)>>1)-64*1024 {
		return int(^uint(0) >> 1)
	}
	return int(maxEntryBytes) + 64*1024
}

func summarizeEntry(entry Entry, file string, lineBytes int64) EntrySummary {
	return EntrySummary{
		ID:         entry.ID,
		RequestID:  entry.RequestID,
		CreatedAt:  entry.CreatedAt,
		Method:     entry.Method,
		Path:       entry.Path,
		Provider:   entry.Provider,
		Model:      entry.Model,
		StatusCode: entry.StatusCode,
		HasError:   strings.TrimSpace(entry.Error) != "",
		File:       file,
		LineBytes:  lineBytes,
	}
}

// RedactHeaders returns a copy with sensitive header values replaced.
func RedactHeaders(headers map[string][]string) map[string][]string {
	if len(headers) == 0 {
		return nil
	}
	out := make(map[string][]string, len(headers))
	for key, values := range headers {
		copied := make([]string, len(values))
		if isSensitiveKey(key) {
			for i := range copied {
				copied[i] = "[REDACTED]"
			}
		} else {
			copy(copied, values)
		}
		out[key] = copied
	}
	return out
}

// RedactMetadata returns a copy with sensitive metadata values replaced.
func RedactMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		if isSensitiveKey(key) {
			out[key] = "[REDACTED]"
			continue
		}
		out[key] = value
	}
	return out
}

// RedactURL redacts URL userinfo and sensitive query values.
func RedactURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}
	if parsed.User != nil {
		parsed.User = url.User("[REDACTED]")
	}
	query := parsed.Query()
	for key := range query {
		if isSensitiveKey(key) {
			query.Set(key, "[REDACTED]")
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

// RedactJSON returns a copy of raw JSON with sensitive object keys replaced.
func RedactJSON(raw json.RawMessage) json.RawMessage {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || !json.Valid([]byte(trimmed)) {
		return cloneRawMessage(raw)
	}
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return cloneRawMessage(raw)
	}
	redacted, err := json.Marshal(redactJSONValue(value))
	if err != nil {
		return cloneRawMessage(raw)
	}
	return json.RawMessage(redacted)
}

func redactJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			if isSensitiveKey(key) {
				out[key] = "[REDACTED]"
				continue
			}
			out[key] = redactJSONValue(child)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = redactJSONValue(child)
		}
		return out
	default:
		return value
	}
}

func cloneRawMessage(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	out := make(json.RawMessage, len(raw))
	copy(out, raw)
	return out
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	if lower == "" {
		return false
	}
	sensitiveFragments := []string{
		"authorization",
		"cookie",
		"password",
		"secret",
		"token",
		"api-key",
		"api_key",
		"apikey",
		"access-key",
		"session-key",
		"credential",
	}
	for _, fragment := range sensitiveFragments {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return lower == "key"
}
