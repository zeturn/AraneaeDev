package executor

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"araneae-go/internal/executor/store"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestFetchAndEmitRSSFetchesArticleContent(t *testing.T) {
	var sawBrowserUA, sawReferer bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/feed.xml":
			w.Header().Set("Content-Type", "application/rss+xml")
			_, _ = w.Write([]byte(`<rss version="2.0"><channel><item>
				<title>Long article</title>
				<link>` + "http://" + r.Host + `/article</link>
				<guid>item-1</guid>
				<pubDate>Sat, 18 Jul 2026 04:00:00 GMT</pubDate>
				<description>Short feed summary.</description>
			</item></channel></rss>`))
		case "/article":
			sawBrowserUA = strings.Contains(r.UserAgent(), "Mozilla/5.0")
			sawReferer = strings.Contains(r.Referer(), "/feed.xml")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html><html><head><title>Ignore</title></head><body>
				<nav>Navigation should not be selected.</nav>
				<article>
					<p>This is the first paragraph of a full article body with enough detail for extraction and storage.</p>
					<p>This is the second paragraph of the article body, adding enough natural language content to pass the minimum length threshold and verify that the extractor keeps multiple paragraphs.</p>
					<p>This is the third paragraph with additional context so the crawler records useful body text instead of only the RSS summary.</p>
				</article>
			</body></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	sinkDir := t.TempDir()
	out, err := fetchAndEmit(context.Background(), "rss", server.URL+"/feed.xml", sinkDir)
	if err != nil {
		t.Fatalf("fetchAndEmit failed: %v", err)
	}
	if !strings.Contains(out, "fetched 1 items") {
		t.Fatalf("unexpected output: %s", out)
	}
	if !sawBrowserUA {
		t.Fatal("article request did not use a browser-like user-agent")
	}
	if !sawReferer {
		t.Fatal("article request did not send feed referer")
	}

	event := readFirstSinkEvent(t, filepath.Join(sinkDir, "events.jsonl"))
	var payload struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		t.Fatalf("decode structured payload: %v", err)
	}
	content, _ := payload.Data["content"].(string)
	if !strings.Contains(content, "full article body") || !strings.Contains(content, "third paragraph") {
		t.Fatalf("expected fetched article body, got: %q", content)
	}
	if status, _ := payload.Data["content_status"].(string); status != "article_fetched" {
		t.Fatalf("expected article_fetched status, got %q", status)
	}
}

func TestFetchAndEmitRSSFallsBackToSummaryWhenArticleBlocked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/feed.xml":
			w.Header().Set("Content-Type", "application/rss+xml")
			_, _ = w.Write([]byte(`<rss version="2.0"><channel><item>
				<title>Blocked article</title>
				<link>` + "http://" + r.Host + `/blocked</link>
				<guid>item-2</guid>
				<description>Summary remains available.</description>
			</item></channel></rss>`))
		case "/blocked":
			http.Error(w, "blocked", http.StatusForbidden)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	sinkDir := t.TempDir()
	if _, err := fetchAndEmit(context.Background(), "rss", server.URL+"/feed.xml", sinkDir); err != nil {
		t.Fatalf("fetchAndEmit failed: %v", err)
	}

	event := readFirstSinkEvent(t, filepath.Join(sinkDir, "events.jsonl"))
	var payload struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		t.Fatalf("decode structured payload: %v", err)
	}
	if content, _ := payload.Data["content"].(string); content != "Summary remains available." {
		t.Fatalf("expected summary fallback, got %q", content)
	}
	if status, _ := payload.Data["content_status"].(string); status != "feed_content" {
		t.Fatalf("expected feed_content status, got %q", status)
	}
}

func TestSourceFetchUsesConditionalRequestAndPersistsValidators(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("If-None-Match") == `"feed-v1"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"feed-v1"`)
		w.Header().Set("Last-Modified", "Sun, 26 Jul 2026 00:00:00 GMT")
		_, _ = w.Write([]byte(`<rss version="2.0"><channel><item><title>Cached item</title><guid>cached-1</guid><description>Summary.</description></item></channel></rss>`))
	}))
	defer server.Close()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "executor.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.AutoMigrate(&store.SourceFetchState{}); err != nil {
		t.Fatal(err)
	}
	app := &App{db: db, log: zap.NewNop()}
	if _, err := app.fetchAndEmitSource(context.Background(), "rss", server.URL, t.TempDir(), "task-cached"); err != nil {
		t.Fatal(err)
	}
	out, err := app.fetchAndEmitSource(context.Background(), "rss", server.URL, t.TempDir(), "task-cached")
	if err != nil {
		t.Fatal(err)
	}
	if out != "source not modified (conditional request)" {
		t.Fatalf("unexpected conditional result: %q", out)
	}
	if requests != 2 {
		t.Fatalf("requests=%d, want 2", requests)
	}
	var state store.SourceFetchState
	if err := db.First(&state, "task_id = ?", "task-cached").Error; err != nil {
		t.Fatal(err)
	}
	if state.ETag != `"feed-v1"` || state.ConsecutiveFailures != 0 || state.LastStatus != http.StatusNotModified {
		t.Fatalf("unexpected source state: %#v", state)
	}
}

func readFirstSinkEvent(t *testing.T, path string) sinkEnvelope {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open sink file: %v", err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		t.Fatalf("missing sink event: %v", scanner.Err())
	}
	var event sinkEnvelope
	if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
		t.Fatalf("decode sink event: %v", err)
	}
	return event
}
