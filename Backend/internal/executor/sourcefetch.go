package executor

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"araneae-go/internal/executor/contracts"
)

// maxSourceFetchBytes caps the response body we are willing to read for rss/api tasks.
const maxSourceFetchBytes = 20 * 1024 * 1024

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

	out, err := fetchAndEmit(runCtx, taskType, sourceURL, sinkDir)
	if err != nil {
		return out, 1, runDir, err
	}
	return out, 0, runDir, nil
}

func fetchAndEmit(ctx context.Context, taskType, sourceURL, sinkDir string) (string, error) {
	body, err := httpGetBytes(ctx, sourceURL)
	if err != nil {
		return "", err
	}
	switch taskType {
	case "rss":
		n, ferr := emitRSS(body, sourceURL, sinkDir)
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

func httpGetBytes(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Araneae Source Fetcher/1.0")
	req.Header.Set("Accept", "application/json, application/xml, text/xml, */*")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch %s failed: status %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxSourceFetchBytes))
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
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title   string     `xml:"title"`
	Links   []atomLink `xml:"link"`
	Summary string     `xml:"summary"`
	Content string     `xml:"content"`
	ID      string     `xml:"id"`
	Updated string     `xml:"updated"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

func emitRSS(raw []byte, sourceURL, sinkDir string) (int, error) {
	if v, err := tryEmitRSS2(raw, sourceURL, sinkDir); err == nil {
		return v, nil
	}
	if v, err := tryEmitAtom(raw, sourceURL, sinkDir); err == nil {
		return v, nil
	}
	if v, err := tryEmitRDF(raw, sourceURL, sinkDir); err == nil {
		return v, nil
	}
	return 0, fmt.Errorf("unrecognized feed format (not RSS 2.0 / Atom / RDF)")
}

func tryEmitRSS2(raw []byte, sourceURL, sinkDir string) (int, error) {
	var f rss2Feed
	if err := xml.Unmarshal(raw, &f); err != nil || len(f.Channel.Items) == 0 {
		return 0, fmt.Errorf("not rss2")
	}
	count := 0
	for _, it := range f.Channel.Items {
		data := map[string]interface{}{
			"title":        strings.TrimSpace(it.Title),
			"link":         strings.TrimSpace(it.Link),
			"summary":      strings.TrimSpace(it.Description),
			"published_at": strings.TrimSpace(it.PubDate),
			"source_url":   sourceURL,
		}
		if err := emitStructured(sinkDir, sourceURL, "rss_item", data); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func tryEmitAtom(raw []byte, sourceURL, sinkDir string) (int, error) {
	var f atomFeed
	if err := xml.Unmarshal(raw, &f); err != nil || len(f.Entries) == 0 {
		return 0, fmt.Errorf("not atom")
	}
	count := 0
	for _, e := range f.Entries {
		link := strings.TrimSpace(e.ID)
		for _, l := range e.Links {
			if l.Rel == "alternate" || l.Rel == "" {
				if h := strings.TrimSpace(l.Href); h != "" {
					link = h
					break
				}
			}
		}
		data := map[string]interface{}{
			"title":        strings.TrimSpace(e.Title),
			"link":         link,
			"summary":      strings.TrimSpace(e.Summary),
			"content":      strings.TrimSpace(e.Content),
			"published_at": strings.TrimSpace(e.Updated),
			"source_url":   sourceURL,
		}
		if err := emitStructured(sinkDir, sourceURL, "rss_item", data); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func tryEmitRDF(raw []byte, sourceURL, sinkDir string) (int, error) {
	var f rdfFeed
	if err := xml.Unmarshal(raw, &f); err != nil || len(f.Items) == 0 {
		return 0, fmt.Errorf("not rdf")
	}
	count := 0
	for _, it := range f.Items {
		data := map[string]interface{}{
			"title":        strings.TrimSpace(it.Title),
			"link":         strings.TrimSpace(it.Link),
			"summary":      strings.TrimSpace(it.Description),
			"published_at": strings.TrimSpace(it.PubDate),
			"source_url":   sourceURL,
		}
		if err := emitStructured(sinkDir, sourceURL, "rss_item", data); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
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
