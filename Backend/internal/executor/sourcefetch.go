package executor

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"araneae-go/internal/executor/contracts"
	"araneae-go/internal/executor/store"

	"go.uber.org/zap"
	"golang.org/x/net/html"
	"gorm.io/gorm"
)

// maxSourceFetchBytes caps the response body we are willing to read for rss/api tasks.
const maxSourceFetchBytes = 20 * 1024 * 1024
const maxArticleFetchBytes = 8 * 1024 * 1024
const articleFetchDelay = 1500 * time.Millisecond
const articleFetchTimeout = 25 * time.Second
const articleContentMinChars = 240
const articleContentMaxChars = 80_000

var whitespaceRE = regexp.MustCompile(`\s+`)

type httpFetchOptions struct {
	Accept       string
	Referer      string
	MaxBytes     int64
	Timeout      time.Duration
	Browserish   bool
	RetryEnabled bool
}

// sourceFetchOptions is carried in task metadata so each source task can set a
// bounded, source-appropriate fetch policy without making a service-wide rule.
// A zero MaxItems preserves the legacy "all feed items" behavior.
type sourceFetchOptions struct {
	MaxItems           int
	FetchArticleBodies bool
}

func defaultSourceFetchOptions() sourceFetchOptions {
	return sourceFetchOptions{FetchArticleBodies: true}
}

func sourceFetchOptionsFromMetadata(metadata map[string]any) sourceFetchOptions {
	opts := defaultSourceFetchOptions()
	raw, ok := metadata["source_fetch"].(map[string]any)
	if !ok {
		return opts
	}
	if value, ok := sourceFetchInt(raw["max_items"]); ok {
		opts.MaxItems = value
	}
	if value, ok := raw["fetch_article_bodies"].(bool); ok {
		opts.FetchArticleBodies = value
	}
	return opts
}

func sourceFetchInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return max(0, typed), true
	case int64:
		return max(0, int(typed)), true
	case float64:
		return max(0, int(typed)), true
	default:
		return 0, false
	}
}

// executeSourceFetch handles non-code tasks (rss / json api). It does not download
// an artifact; instead it fetches the source URL and emits each record into the sink
// directory so the existing sink pipeline forwards them to HashSlip.
func (a *App) executeSourceFetch(ctx context.Context, taskType, sourceURL string, msg contracts.QueueTaskMessage) (string, int, string, error) {
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(a.cfg.TaskTimeoutSeconds)*time.Second)
	defer cancel()

	runDir := filepath.Join(a.cfg.WorkDir, msg.RunID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return "", 1, runDir, err
	}
	sinkDir := filepath.Join(runDir, ".araneae", "sink")
	if err := os.MkdirAll(sinkDir, 0o755); err != nil {
		return "", 1, runDir, err
	}

	out, err := a.fetchAndEmitSource(runCtx, taskType, sourceURL, sinkDir, msg.TaskID, sourceFetchOptionsFromMetadata(msg.Metadata))
	if err != nil {
		return out, 1, runDir, err
	}
	return out, 0, runDir, nil
}

func fetchAndEmit(ctx context.Context, taskType, sourceURL, sinkDir string) (string, error) {
	result, err := httpGetBytesWithHeaders(ctx, sourceURL, httpFetchOptions{
		Accept:       "application/json, application/xml, text/xml, */*",
		MaxBytes:     maxSourceFetchBytes,
		Timeout:      30 * time.Second,
		RetryEnabled: true,
	}, nil)
	if err != nil {
		return "", err
	}
	return emitSourceBody(ctx, taskType, sourceURL, sinkDir, result.Body, defaultSourceFetchOptions())
}

func (a *App) fetchAndEmitSource(ctx context.Context, taskType, sourceURL, sinkDir, taskID string, options sourceFetchOptions) (string, error) {
	now := time.Now().UTC()
	state := store.SourceFetchState{TaskID: taskID}
	if err := a.db.Where("task_id = ?", taskID).First(&state).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", fmt.Errorf("load source fetch state: %w", err)
	}
	if !state.NextAttemptAt.IsZero() && now.Before(state.NextAttemptAt) {
		return fmt.Sprintf("source fetch deferred until %s after %d consecutive failures", state.NextAttemptAt.Format(time.RFC3339), state.ConsecutiveFailures), nil
	}

	headers := map[string]string{}
	if state.ETag != "" {
		headers["If-None-Match"] = state.ETag
	}
	if state.LastModified != "" {
		headers["If-Modified-Since"] = state.LastModified
	}
	result, err := httpGetBytesWithHeaders(ctx, sourceURL, httpFetchOptions{
		Accept:       "application/json, application/xml, text/xml, */*",
		MaxBytes:     maxSourceFetchBytes,
		Timeout:      30 * time.Second,
		RetryEnabled: true,
	}, headers)
	if err != nil {
		a.recordSourceFetchFailure(state, now, err)
		return "", err
	}
	if result.NotModified {
		a.recordSourceFetchSuccess(state, now, result)
		return "source not modified (conditional request)", nil
	}
	out, err := emitSourceBody(ctx, taskType, sourceURL, sinkDir, result.Body, options)
	if err != nil {
		a.recordSourceFetchFailure(state, now, err)
		return out, err
	}
	a.recordSourceFetchSuccess(state, now, result)
	return out, nil
}

func emitSourceBody(ctx context.Context, taskType, sourceURL, sinkDir string, body []byte, options sourceFetchOptions) (string, error) {
	switch taskType {
	case "rss":
		n, ferr := emitRSS(ctx, body, sourceURL, sinkDir, options)
		if ferr != nil {
			return "", ferr
		}
		return fmt.Sprintf("fetched %d items from RSS/Atom feed", n), nil
	case "api":
		n, ferr := emitJSONAPI(body, sourceURL, sinkDir)
		if ferr != nil {
			return "", ferr
		}
		return fmt.Sprintf("fetched JSON API: %d records", n), nil
	}
	return "", fmt.Errorf("unsupported source type: %s", taskType)
}

func httpGetBytes(ctx context.Context, targetURL string, opts httpFetchOptions) ([]byte, error) {
	result, err := httpGetBytesWithHeaders(ctx, targetURL, opts, nil)
	return result.Body, err
}

type httpFetchResult struct {
	Body         []byte
	StatusCode   int
	ETag         string
	LastModified string
	NotModified  bool
}

func httpGetBytesWithHeaders(ctx context.Context, targetURL string, opts httpFetchOptions, headers map[string]string) (httpFetchResult, error) {
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = maxSourceFetchBytes
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	attempts := 1
	if opts.RetryEnabled {
		attempts = 3
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			if err := sleepContext(ctx, time.Duration(attempt)*750*time.Millisecond); err != nil {
				return httpFetchResult{}, err
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
		if err != nil {
			return httpFetchResult{}, err
		}
		applyCrawlerHeaders(req, opts)
		for name, value := range headers {
			if strings.TrimSpace(value) != "" {
				req.Header.Set(name, value)
			}
		}
		client := &http.Client{Timeout: opts.Timeout}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode == http.StatusNotModified {
			resp.Body.Close()
			return httpFetchResult{StatusCode: resp.StatusCode, ETag: resp.Header.Get("ETag"), LastModified: resp.Header.Get("Last-Modified"), NotModified: true}, nil
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, opts.MaxBytes+1))
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if int64(len(body)) > opts.MaxBytes {
			return httpFetchResult{}, fmt.Errorf("fetch %s failed: response too large (max %d bytes)", targetURL, opts.MaxBytes)
		}
		if resp.StatusCode < 300 {
			return httpFetchResult{Body: body, StatusCode: resp.StatusCode, ETag: resp.Header.Get("ETag"), LastModified: resp.Header.Get("Last-Modified")}, nil
		}
		lastErr = fmt.Errorf("fetch %s failed: status %d", targetURL, resp.StatusCode)
		if !retryableStatus(resp.StatusCode) || attempt == attempts-1 {
			return httpFetchResult{}, lastErr
		}
		if wait := retryAfter(resp.Header.Get("Retry-After")); wait > 0 {
			if err := sleepContext(ctx, wait); err != nil {
				return httpFetchResult{}, err
			}
		}
	}
	return httpFetchResult{}, lastErr
}

func (a *App) recordSourceFetchSuccess(state store.SourceFetchState, now time.Time, result httpFetchResult) {
	if result.ETag != "" {
		state.ETag = result.ETag
	}
	if result.LastModified != "" {
		state.LastModified = result.LastModified
	}
	state.ConsecutiveFailures = 0
	state.NextAttemptAt = time.Time{}
	state.LastStatus = result.StatusCode
	state.LastError = ""
	state.UpdatedAt = now
	if err := a.db.Save(&state).Error; err != nil {
		a.log.Warn("persist source fetch success state failed", zap.Error(err), zap.String("task_id", state.TaskID))
	}
}

func (a *App) recordSourceFetchFailure(state store.SourceFetchState, now time.Time, fetchErr error) {
	state.ConsecutiveFailures++
	backoff := 15 * time.Minute * time.Duration(1<<min(state.ConsecutiveFailures-1, 5))
	if backoff > 6*time.Hour {
		backoff = 6 * time.Hour
	}
	state.NextAttemptAt = now.Add(backoff)
	state.LastStatus = 0
	state.LastError = truncateRunes(fetchErr.Error(), 2048)
	state.UpdatedAt = now
	if err := a.db.Save(&state).Error; err != nil {
		a.log.Warn("persist source fetch failure state failed", zap.Error(err), zap.String("task_id", state.TaskID))
	}
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func applyCrawlerHeaders(req *http.Request, opts httpFetchOptions) {
	ua := "Araneae Source Fetcher/1.0"
	if opts.Browserish {
		ua = browserUserAgent(req.URL.String())
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Pragma", "no-cache")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Site", "cross-site")
	}
	req.Header.Set("User-Agent", ua)
	if opts.Accept != "" {
		req.Header.Set("Accept", opts.Accept)
	} else {
		req.Header.Set("Accept", "*/*")
	}
	if opts.Referer != "" {
		req.Header.Set("Referer", opts.Referer)
	}
}

func browserUserAgent(key string) string {
	agents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0 Safari/537.36",
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return agents[int(h.Sum32())%len(agents)]
}

func retryableStatus(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status >= 500
}

func retryAfter(raw string) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if seconds, err := time.ParseDuration(raw + "s"); err == nil {
		if seconds > 10*time.Second {
			return 10 * time.Second
		}
		return seconds
	}
	if when, err := http.ParseTime(raw); err == nil {
		wait := time.Until(when)
		if wait > 10*time.Second {
			return 10 * time.Second
		}
		if wait > 0 {
			return wait
		}
	}
	return 0
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func emitStructured(sinkDir, instanceID, schemaID string, data map[string]interface{}) error {
	payload := map[string]interface{}{
		"instance_id": instanceID,
		"schema_id":   schemaID,
		"data":        data,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	env := sinkEnvelope{
		Type:      "structured",
		Data:      raw,
		EmittedAt: time.Now().UTC().Format(time.RFC3339),
	}
	line, err := json.Marshal(env)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(sinkDir, "events.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}

// ---- RSS / Atom / RDF parsing ----

type rss2Feed struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rdfFeed struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string        `xml:"title"`
	Links       []rssItemLink `xml:"link"`
	Description string        `xml:"description"`
	PubDate     string        `xml:"pubDate"`
	GUID        string        `xml:"guid"`
}

type rssItemLink struct {
	XMLName xml.Name `xml:"link"`
	Href    string   `xml:"href,attr"`
	Text    string   `xml:",chardata"`
}

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title     string     `xml:"title"`
	Links     []atomLink `xml:"link"`
	Summary   string     `xml:"summary"`
	Content   string     `xml:"content"`
	ID        string     `xml:"id"`
	Published string     `xml:"published"`
	Updated   string     `xml:"updated"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

func emitRSS(ctx context.Context, raw []byte, sourceURL, sinkDir string, options sourceFetchOptions) (int, error) {
	if v, err := tryEmitRSS2(ctx, raw, sourceURL, sinkDir, options); err == nil {
		return v, nil
	}
	if v, err := tryEmitAtom(ctx, raw, sourceURL, sinkDir, options); err == nil {
		return v, nil
	}
	if v, err := tryEmitRDF(ctx, raw, sourceURL, sinkDir, options); err == nil {
		return v, nil
	}
	return 0, fmt.Errorf("unrecognized feed format (not RSS 2.0 / Atom / RDF)")
}

func tryEmitRSS2(ctx context.Context, raw []byte, sourceURL, sinkDir string, options sourceFetchOptions) (int, error) {
	var f rss2Feed
	if err := xml.Unmarshal(raw, &f); err != nil || len(f.Channel.Items) == 0 {
		return 0, fmt.Errorf("not rss2")
	}
	count := 0
	for index, it := range f.Channel.Items {
		if options.MaxItems > 0 && index >= options.MaxItems {
			break
		}
		link := bestRSSItemLink(it.Links)
		rawPublishedAt := strings.TrimSpace(it.PubDate)
		publishedAt := normalizePublishedAt(rawPublishedAt)
		summary := strings.TrimSpace(it.Description)
		content, contentStatus := fetchArticleContentWithOptions(ctx, link, sourceURL, summary, options)
		data := map[string]interface{}{
			"title":                   strings.TrimSpace(it.Title),
			"link":                    link,
			"summary":                 summary,
			"content":                 content,
			"content_status":          contentStatus,
			"published_at":            publishedAt,
			"source_published_at_raw": rawPublishedAt,
			"source_url":              sourceURL,
			"id":                      sourceItemID(strings.TrimSpace(it.GUID), link, strings.TrimSpace(it.Title), rawPublishedAt+"|published_at_v1"),
		}
		if err := emitStructured(sinkDir, sourceURL, "rss_item", data); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func tryEmitAtom(ctx context.Context, raw []byte, sourceURL, sinkDir string, options sourceFetchOptions) (int, error) {
	var f atomFeed
	if err := xml.Unmarshal(raw, &f); err != nil || len(f.Entries) == 0 {
		return 0, fmt.Errorf("not atom")
	}
	count := 0
	for index, e := range f.Entries {
		if options.MaxItems > 0 && index >= options.MaxItems {
			break
		}
		link := strings.TrimSpace(e.ID)
		for _, l := range e.Links {
			if l.Rel == "alternate" || l.Rel == "" {
				if h := strings.TrimSpace(l.Href); h != "" {
					link = h
					break
				}
			}
		}
		rawPublishedAt := strings.TrimSpace(e.Published)
		if rawPublishedAt == "" {
			rawPublishedAt = strings.TrimSpace(e.Updated)
		}
		publishedAt := normalizePublishedAt(rawPublishedAt)
		feedContent := strings.TrimSpace(e.Content)
		content, contentStatus := fetchArticleContentWithOptions(ctx, link, sourceURL, feedContent, options)
		data := map[string]interface{}{
			"title":                   strings.TrimSpace(e.Title),
			"link":                    link,
			"summary":                 strings.TrimSpace(e.Summary),
			"content":                 content,
			"content_status":          contentStatus,
			"published_at":            publishedAt,
			"source_published_at_raw": rawPublishedAt,
			"source_url":              sourceURL,
			"id":                      sourceItemID(strings.TrimSpace(e.ID), link, strings.TrimSpace(e.Title), rawPublishedAt+"|published_at_v1"),
		}
		if err := emitStructured(sinkDir, sourceURL, "rss_item", data); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func tryEmitRDF(ctx context.Context, raw []byte, sourceURL, sinkDir string, options sourceFetchOptions) (int, error) {
	var f rdfFeed
	if err := xml.Unmarshal(raw, &f); err != nil || len(f.Items) == 0 {
		return 0, fmt.Errorf("not rdf")
	}
	count := 0
	for index, it := range f.Items {
		if options.MaxItems > 0 && index >= options.MaxItems {
			break
		}
		link := bestRSSItemLink(it.Links)
		rawPublishedAt := strings.TrimSpace(it.PubDate)
		publishedAt := normalizePublishedAt(rawPublishedAt)
		summary := strings.TrimSpace(it.Description)
		content, contentStatus := fetchArticleContentWithOptions(ctx, link, sourceURL, summary, options)
		data := map[string]interface{}{
			"title":                   strings.TrimSpace(it.Title),
			"link":                    link,
			"summary":                 summary,
			"content":                 content,
			"content_status":          contentStatus,
			"published_at":            publishedAt,
			"source_published_at_raw": rawPublishedAt,
			"source_url":              sourceURL,
			"id":                      sourceItemID(strings.TrimSpace(it.GUID), link, strings.TrimSpace(it.Title), rawPublishedAt+"|published_at_v1"),
		}
		if err := emitStructured(sinkDir, sourceURL, "rss_item", data); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// normalizePublishedAt gives downstream rolling windows one canonical time
// representation while retaining the source string separately for audit.  RSS
// feeds commonly use RFC 822/1123 variants, whereas Atom normally uses
// RFC3339.  An unparseable timestamp is intentionally returned as empty: a
// time-bounded Brief must not silently treat an undated item as current.
func normalizePublishedAt(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.RFC850,
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05 -0700",
	} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.UTC().Format(time.RFC3339)
		}
	}
	return ""
}

func fetchArticleContent(ctx context.Context, articleURL, referer, fallback string) (string, string) {
	return fetchArticleContentWithOptions(ctx, articleURL, referer, fallback, defaultSourceFetchOptions())
}

func fetchArticleContentWithOptions(ctx context.Context, articleURL, referer, fallback string, options sourceFetchOptions) (string, string) {
	fallback = cleanText(fallback)
	if !options.FetchArticleBodies {
		if fallback != "" {
			return fallback, "feed_content"
		}
		return "", "article_body_disabled"
	}
	if articleURL == "" {
		if fallback != "" {
			return fallback, "feed_content"
		}
		return "", "missing_link"
	}
	parsed, err := url.Parse(articleURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		if fallback != "" {
			return fallback, "feed_content"
		}
		return "", "invalid_link"
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		if fallback != "" {
			return fallback, "feed_content"
		}
		return "", "unsupported_link_scheme"
	}

	if err := sleepContext(ctx, articleFetchDelay); err != nil {
		if fallback != "" {
			return fallback, "feed_content"
		}
		return "", "article_fetch_cancelled"
	}
	raw, err := httpGetBytes(ctx, articleURL, httpFetchOptions{
		Accept:       "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		Referer:      referer,
		MaxBytes:     maxArticleFetchBytes,
		Timeout:      articleFetchTimeout,
		Browserish:   true,
		RetryEnabled: true,
	})
	if err != nil {
		if fallback != "" {
			return fallback, "feed_content"
		}
		return "", "article_fetch_failed: " + err.Error()
	}
	content, err := extractArticleText(raw)
	if err != nil || len([]rune(content)) < articleContentMinChars {
		if fallback != "" {
			return fallback, "feed_content"
		}
		if err != nil {
			return "", "article_extract_failed: " + err.Error()
		}
		return "", "article_extract_too_short"
	}
	return truncateRunes(content, articleContentMaxChars), "article_fetched"
}

func extractArticleText(raw []byte) (string, error) {
	root, err := html.Parse(strings.NewReader(string(raw)))
	if err != nil {
		return "", err
	}
	var candidates []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			tag := strings.ToLower(n.Data)
			if tag == "script" || tag == "style" || tag == "noscript" || tag == "template" || tag == "svg" {
				return
			}
			if tag == "article" || tag == "main" || hasArticleClass(n) {
				if text := cleanText(textFromNode(n)); text != "" {
					candidates = append(candidates, text)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)

	best := ""
	for _, c := range candidates {
		if len([]rune(c)) > len([]rune(best)) {
			best = c
		}
	}
	if len([]rune(best)) >= articleContentMinChars {
		return best, nil
	}

	var paragraphs []string
	collectParagraphs(root, &paragraphs)
	best = cleanText(strings.Join(paragraphs, "\n\n"))
	if best == "" {
		return "", fmt.Errorf("no readable article text")
	}
	return best, nil
}

func hasArticleClass(n *html.Node) bool {
	for _, attr := range n.Attr {
		key := strings.ToLower(attr.Key)
		if key != "class" && key != "id" && key != "role" {
			continue
		}
		value := strings.ToLower(attr.Val)
		for _, token := range []string{"article", "story", "entry-content", "post-content", "content-body"} {
			if strings.Contains(value, token) {
				return true
			}
		}
	}
	return false
}

func collectParagraphs(n *html.Node, out *[]string) {
	if n.Type == html.ElementNode {
		tag := strings.ToLower(n.Data)
		if tag == "script" || tag == "style" || tag == "noscript" || tag == "template" || tag == "svg" ||
			tag == "nav" || tag == "footer" || tag == "header" || tag == "aside" {
			return
		}
		if tag == "p" {
			text := cleanText(textFromNode(n))
			if len([]rune(text)) >= 40 {
				*out = append(*out, text)
			}
			return
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectParagraphs(c, out)
	}
}

func textFromNode(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
			b.WriteByte(' ')
			return
		}
		if node.Type == html.ElementNode {
			tag := strings.ToLower(node.Data)
			if tag == "script" || tag == "style" || tag == "noscript" || tag == "template" || tag == "svg" {
				return
			}
			if tag == "p" || tag == "br" || tag == "div" || tag == "section" || tag == "li" || tag == "h1" || tag == "h2" || tag == "h3" {
				b.WriteByte('\n')
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func cleanText(s string) string {
	lines := strings.Split(s, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(whitespaceRE.ReplaceAllString(line, " "))
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	return strings.Join(cleaned, "\n\n")
}

func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func bestRSSItemLink(links []rssItemLink) string {
	for _, link := range links {
		if link.XMLName.Space == "" {
			if text := strings.TrimSpace(link.Text); text != "" {
				return text
			}
		}
	}
	for _, link := range links {
		if href := strings.TrimSpace(link.Href); href != "" {
			return href
		}
	}
	for _, link := range links {
		if text := strings.TrimSpace(link.Text); text != "" {
			return text
		}
	}
	return ""
}

func sourceItemID(preferred, link, title, publishedAt string) string {
	if preferred != "" {
		return preferred
	}
	if link != "" {
		return link
	}
	return title + "\n" + publishedAt
}

// ---- JSON API parsing ----

func emitJSONAPI(raw []byte, sourceURL, sinkDir string) (int, error) {
	var anyJSON interface{}
	if err := json.Unmarshal(raw, &anyJSON); err != nil {
		return 0, fmt.Errorf("invalid JSON response: %w", err)
	}
	records := []interface{}{}
	switch v := anyJSON.(type) {
	case []interface{}:
		records = v
	case map[string]interface{}:
		records = []interface{}{v}
	default:
		return 0, fmt.Errorf("unsupported JSON top-level type: %T", anyJSON)
	}
	count := 0
	for _, rec := range records {
		data, ok := rec.(map[string]interface{})
		if !ok {
			data = map[string]interface{}{"value": rec}
		}
		if err := emitStructured(sinkDir, sourceURL, "json_api_record", data); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
