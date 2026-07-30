// source-catalog-import validates and applies a versioned source catalog to an
// Araneae control plane. It deliberately contains no news-specific behavior:
// source type, cadence, task metadata and destination slot are input data.
package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/html/charset"
)

type catalog struct {
	ID       string          `json:"id"`
	Version  string          `json:"version"`
	Defaults catalogDefaults `json:"defaults"`
	Sources  []catalogSource `json:"sources"`
}

type catalogDefaults struct {
	TaskType  string `json:"task_type"`
	Schedule  string `json:"schedule"`
	NodeQueue string `json:"node_queue"`
}

type catalogSource struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Category           string `json:"category"`
	URL                string `json:"url"`
	TaskType           string `json:"task_type,omitempty"`
	Schedule           string `json:"schedule,omitempty"`
	NodeQueue          string `json:"node_queue,omitempty"`
	MaxItems           *int   `json:"max_items,omitempty"`
	FetchArticleBodies *bool  `json:"fetch_article_bodies,omitempty"`
	JSONRecordPath     string `json:"json_record_path,omitempty"`
	Enabled            *bool  `json:"enabled,omitempty"`
}

type task struct {
	ID        string `json:"id"`
	SourceURL string `json:"source_url"`
	Metadata  string `json:"metadata"`
}

type schedule struct {
	ID     string `json:"id"`
	TaskID string `json:"task_id"`
}

func main() {
	catalogPath := flag.String("catalog", "", "path to a catalog JSON file")
	apiBase := flag.String("api-base", "", "Araneae API base URL, for example https://araneae.example.com/api/v1")
	token := flag.String("token", os.Getenv("ARANEAE_ACCESS_TOKEN"), "Araneae bearer token (or ARANEAE_ACCESS_TOKEN)")
	datasetID := flag.String("dataset-id", "", "HashSlip dataset id for structured source records")
	missionID := flag.String("mission-id", "", "optional Orion mission id stored in task metadata")
	apply := flag.Bool("apply", false, "create missing tasks and schedules after validation")
	skipVerify := flag.Bool("skip-verify", false, "do not make HTTP feed validation requests")
	timeout := flag.Duration("timeout", 20*time.Second, "per-source HTTP validation timeout")
	flag.Parse()

	if strings.TrimSpace(*catalogPath) == "" {
		fatal(errors.New("-catalog is required"))
	}
	value, err := loadCatalog(*catalogPath)
	if err != nil {
		fatal(err)
	}
	if !*skipVerify {
		if err := verifyCatalog(value, *timeout); err != nil {
			fatal(err)
		}
	}
	if !*apply {
		fmt.Printf("catalog %s@%s validated: %d source(s)\n", value.ID, value.Version, len(value.Sources))
		return
	}
	if strings.TrimSpace(*apiBase) == "" || strings.TrimSpace(*token) == "" || strings.TrimSpace(*datasetID) == "" {
		fatal(errors.New("-apply requires -api-base, -token/ARANEAE_ACCESS_TOKEN and -dataset-id"))
	}
	client := &controlClient{baseURL: strings.TrimRight(*apiBase, "/"), token: *token, http: &http.Client{Timeout: 20 * time.Second}}
	if err := applyCatalog(client, value, *datasetID, *missionID); err != nil {
		fatal(err)
	}
}

func loadCatalog(path string) (catalog, error) {
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return catalog{}, err
	}
	var value catalog
	if err := json.Unmarshal(raw, &value); err != nil {
		return catalog{}, fmt.Errorf("decode catalog: %w", err)
	}
	if strings.TrimSpace(value.ID) == "" || strings.TrimSpace(value.Version) == "" || len(value.Sources) == 0 {
		return catalog{}, errors.New("catalog id, version and sources are required")
	}
	if value.Defaults.TaskType == "" {
		value.Defaults.TaskType = "rss"
	}
	if value.Defaults.Schedule == "" {
		return catalog{}, errors.New("catalog defaults.schedule is required")
	}
	if value.Defaults.NodeQueue == "" {
		value.Defaults.NodeQueue = "default"
	}
	seen := map[string]bool{}
	for index := range value.Sources {
		source := &value.Sources[index]
		source.ID = strings.TrimSpace(source.ID)
		source.Name = strings.TrimSpace(source.Name)
		source.Category = strings.TrimSpace(source.Category)
		source.URL = strings.TrimSpace(source.URL)
		if source.ID == "" || source.Name == "" || source.Category == "" || source.URL == "" {
			return catalog{}, fmt.Errorf("sources[%d] requires id, name, category and url", index)
		}
		if seen[source.ID] {
			return catalog{}, fmt.Errorf("duplicate source id %q", source.ID)
		}
		seen[source.ID] = true
		source.TaskType = strings.ToLower(strings.TrimSpace(source.TaskType))
		if source.TaskType == "" {
			source.TaskType = value.Defaults.TaskType
		}
		if source.TaskType != "rss" && source.TaskType != "api" && source.TaskType != "page" {
			return catalog{}, fmt.Errorf("sources[%d] has invalid task_type %q", index, source.TaskType)
		}
		source.Schedule = strings.TrimSpace(source.Schedule)
		source.NodeQueue = strings.TrimSpace(source.NodeQueue)
		source.JSONRecordPath = strings.TrimSpace(source.JSONRecordPath)
		parsed, err := url.Parse(source.URL)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
			return catalog{}, fmt.Errorf("sources[%d] has invalid URL %q", index, source.URL)
		}
	}
	return value, nil
}

func verifyCatalog(value catalog, timeout time.Duration) error {
	client := &http.Client{Timeout: timeout}
	failures := make([]string, 0)
	for _, source := range value.Sources {
		if source.Enabled != nil && !*source.Enabled {
			continue
		}
		if err := verifySource(client, source); err != nil {
			failures = append(failures, source.ID+": "+err.Error())
			continue
		}
		fmt.Printf("verified %s %s\n", source.ID, source.URL)
	}
	if len(failures) > 0 {
		sort.Strings(failures)
		return fmt.Errorf("catalog verification failed for %d source(s): %s", len(failures), strings.Join(failures, "; "))
	}
	return nil
}

func verifySource(client *http.Client, source catalogSource) error {
	switch source.TaskType {
	case "rss":
		return verifyFeed(client, source.URL)
	case "api":
		return verifyJSONAPI(client, source.URL)
	case "page":
		return verifyPage(client, source.URL)
	default:
		return fmt.Errorf("unsupported task_type %q", source.TaskType)
	}
}

func verifyFeed(client *http.Client, sourceURL string) error {
	req, err := http.NewRequest(http.MethodGet, sourceURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml, */*")
	req.Header.Set("User-Agent", "Araneae Source Catalog Verifier/1.0")
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", response.StatusCode)
	}
	decoder := xml.NewDecoder(io.LimitReader(response.Body, 128*1024))
	decoder.CharsetReader = charset.NewReaderLabel
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return errors.New("empty XML document")
		}
		if err != nil {
			return fmt.Errorf("invalid XML: %w", err)
		}
		if start, ok := token.(xml.StartElement); ok {
			switch strings.ToLower(start.Name.Local) {
			case "rss", "feed", "rdf":
				return nil
			default:
				return fmt.Errorf("unexpected root element %q", start.Name.Local)
			}
		}
	}
}

func verifyJSONAPI(client *http.Client, sourceURL string) error {
	req, err := http.NewRequest(http.MethodGet, sourceURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json, */*")
	req.Header.Set("User-Agent", "Araneae Source Catalog Verifier/1.0")
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", response.StatusCode)
	}
	var value any
	if err := json.NewDecoder(io.LimitReader(response.Body, 8*1024*1024)).Decode(&value); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func verifyPage(client *http.Client, sourceURL string) error {
	req, err := http.NewRequest(http.MethodGet, sourceURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Araneae Source Catalog Verifier/1.0)")
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", response.StatusCode)
	}
	decoder := xml.NewDecoder(io.LimitReader(response.Body, 128*1024))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return errors.New("empty page")
		}
		if err != nil {
			return nil
		}
		if start, ok := token.(xml.StartElement); ok {
			if strings.EqualFold(start.Name.Local, "html") {
				return nil
			}
			return fmt.Errorf("unexpected root element %q", start.Name.Local)
		}
	}
}

type controlClient struct {
	baseURL string
	token   string
	http    *http.Client
}

func (c *controlClient) get(path string, result any) error {
	request, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(request, result)
}

func (c *controlClient) post(path string, body any, result any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	return c.do(request, result)
}

func (c *controlClient) do(request *http.Request, result any) error {
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Accept", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%s %s: HTTP %d: %s", request.Method, request.URL.Path, response.StatusCode, strings.TrimSpace(string(raw)))
	}
	if result != nil && len(raw) > 0 {
		return json.Unmarshal(raw, result)
	}
	return nil
}

func applyCatalog(client *controlClient, value catalog, datasetID, missionID string) error {
	var tasks []task
	if err := client.get("/tasks", &tasks); err != nil {
		return err
	}
	var schedules []schedule
	if err := client.get("/schedules", &schedules); err != nil {
		return err
	}
	existingTasks := map[string]task{}
	for _, item := range tasks {
		var metadata map[string]any
		if json.Unmarshal([]byte(item.Metadata), &metadata) != nil {
			continue
		}
		catalogMeta, _ := metadata["source_catalog"].(map[string]any)
		if fmt.Sprint(catalogMeta["id"]) == value.ID && fmt.Sprint(catalogMeta["version"]) == value.Version {
			existingTasks[fmt.Sprint(catalogMeta["source_id"])] = item
		}
	}
	existingSchedules := map[string]bool{}
	for _, item := range schedules {
		existingSchedules[item.TaskID] = true
	}
	createdTasks, createdSchedules := 0, 0
	for _, source := range value.Sources {
		if source.Enabled != nil && !*source.Enabled {
			continue
		}
		item, exists := existingTasks[source.ID]
		if exists && item.SourceURL != source.URL {
			return fmt.Errorf("catalog task %s URL drift: existing=%s catalog=%s", source.ID, item.SourceURL, source.URL)
		}
		if !exists {
			taskType := strings.TrimSpace(source.TaskType)
			if taskType == "" {
				taskType = value.Defaults.TaskType
			}
			nodeQueue := strings.TrimSpace(source.NodeQueue)
			if nodeQueue == "" {
				nodeQueue = value.Defaults.NodeQueue
			}
			metadata := map[string]any{
				"source_catalog":  map[string]any{"id": value.ID, "version": value.Version, "source_id": source.ID},
				"source_category": source.Category,
				"hashslip_slot":   map[string]any{"dataset_id": datasetID},
				"schema_id":       "news.article.v1",
				"artifact_type":   "news_article",
			}
			sourceFetch := map[string]any{}
			if source.MaxItems != nil {
				sourceFetch["max_items"] = *source.MaxItems
			}
			if source.FetchArticleBodies != nil {
				sourceFetch["fetch_article_bodies"] = *source.FetchArticleBodies
			}
			if source.JSONRecordPath != "" {
				sourceFetch["json_record_path"] = source.JSONRecordPath
			}
			if len(sourceFetch) > 0 {
				metadata["source_fetch"] = sourceFetch
			}
			if missionID != "" {
				metadata["mission_id"] = missionID
			}
			body := map[string]any{"name": source.Name, "type": taskType, "source_url": source.URL, "metadata": metadata, "node_queue": nodeQueue}
			if err := client.post("/tasks", body, &item); err != nil {
				return fmt.Errorf("create task %s: %w", source.ID, err)
			}
			createdTasks++
		}
		if !existingSchedules[item.ID] {
			schedule := strings.TrimSpace(source.Schedule)
			if schedule == "" {
				schedule = value.Defaults.Schedule
			}
			nodeQueue := strings.TrimSpace(source.NodeQueue)
			if nodeQueue == "" {
				nodeQueue = value.Defaults.NodeQueue
			}
			body := map[string]any{"name": value.ID + "-" + value.Version + "-" + source.ID, "description": "Catalog-managed source schedule", "task_id": item.ID, "trigger_type": "crons", "cron_expr": schedule, "node_queue": nodeQueue, "enabled": true}
			if err := client.post("/schedules", body, nil); err != nil {
				return fmt.Errorf("create schedule %s: %w", source.ID, err)
			}
			createdSchedules++
		}
	}
	fmt.Printf("catalog %s@%s applied: created_tasks=%d created_schedules=%d\n", value.ID, value.Version, createdTasks, createdSchedules)
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "source-catalog-import:", err)
	os.Exit(1)
}
