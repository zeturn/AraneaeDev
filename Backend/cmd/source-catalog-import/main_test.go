package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestApplyCatalogUsesOneHashSlipDatasetPerSource(t *testing.T) {
	value := catalog{
		ID:      "news",
		Version: "v1",
		Defaults: catalogDefaults{
			TaskType:  "rss",
			Schedule:  "17 * * * *",
			NodeQueue: "default",
		},
		Sources: []catalogSource{
			{ID: "source-a", Name: "Source A", Category: "news", URL: "https://example.com/a.xml", TaskType: "rss"},
			{ID: "source-b", Name: "Source B", Category: "news", URL: "https://example.com/b.xml", TaskType: "rss"},
		},
	}

	var createdDatasets []map[string]any
	hashslip := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/datasets/ds_template":
			writeJSON(t, w, map[string]any{
				"id": "ds_template", "chunk_id": "chunk_news", "name": "template", "data_type": "structured",
				"schema":   map[string]any{"type": "object"},
				"identity": map[string]any{"dedupe_fields": []string{"id"}, "hash_algorithm": "sha256"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/data-chunks/chunk_news/datasets":
			writeJSON(t, w, []map[string]any{{
				"id": "ds_b", "chunk_id": "chunk_news", "name": "source-source-b", "data_type": "structured",
				"schema":   map[string]any{"type": "object"},
				"identity": map[string]any{"dedupe_fields": []string{"id"}, "hash_algorithm": "sha256"},
			}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/data-chunks/chunk_news/datasets":
			var input map[string]any
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Fatal(err)
			}
			createdDatasets = append(createdDatasets, input)
			writeJSON(t, w, map[string]any{"id": "ds_a", "chunk_id": "chunk_news", "name": input["name"], "data_type": input["data_type"]})
		default:
			http.NotFound(w, r)
		}
	}))
	defer hashslip.Close()

	resolver, err := newPerSourceDatasetResolver(
		&hashslipClient{baseURL: hashslip.URL, token: "token", http: hashslip.Client()},
		value,
		"ds_template",
		"",
		"source-",
	)
	if err != nil {
		t.Fatal(err)
	}

	existingMeta, _ := json.Marshal(map[string]any{
		"source_catalog":  map[string]any{"id": "news", "version": "v1", "source_id": "source-b"},
		"source_category": "news",
		"hashslip_slot":   map[string]any{"dataset_id": "old_shared"},
		"schema_id":       "news.article.v1",
		"artifact_type":   "news_article",
	})
	var createdTasks, updatedTasks []map[string]any
	control := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/tasks":
			writeJSON(t, w, []task{{ID: "task-b", SourceURL: "https://example.com/b.xml", Metadata: string(existingMeta)}})
		case r.Method == http.MethodGet && r.URL.Path == "/schedules":
			writeJSON(t, w, []schedule{})
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			var input map[string]any
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Fatal(err)
			}
			createdTasks = append(createdTasks, input)
			writeJSON(t, w, task{ID: "task-a", SourceURL: "https://example.com/a.xml"})
		case r.Method == http.MethodPut && r.URL.Path == "/tasks/task-b":
			var input map[string]any
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Fatal(err)
			}
			updatedTasks = append(updatedTasks, input)
			writeJSON(t, w, task{ID: "task-b", SourceURL: "https://example.com/b.xml"})
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/schedules"):
			w.WriteHeader(http.StatusCreated)
		default:
			http.NotFound(w, r)
		}
	}))
	defer control.Close()

	client := &controlClient{baseURL: control.URL, token: "token", http: control.Client()}
	if err := applyCatalog(client, value, resolver, "mission-news"); err != nil {
		t.Fatal(err)
	}
	if len(createdDatasets) != 1 {
		t.Fatalf("created datasets=%d, want 1", len(createdDatasets))
	}
	if len(createdTasks) != 1 || len(updatedTasks) != 1 {
		t.Fatalf("created tasks=%d updated tasks=%d, want 1/1", len(createdTasks), len(updatedTasks))
	}
	createdMeta := createdTasks[0]["metadata"].(map[string]any)
	createdSlot := createdMeta["hashslip_slot"].(map[string]any)
	if createdSlot["dataset_id"] != "ds_a" {
		t.Fatalf("created task dataset_id=%v, want ds_a", createdSlot["dataset_id"])
	}
	updatedMeta := updatedTasks[0]["metadata"].(map[string]any)
	updatedSlot := updatedMeta["hashslip_slot"].(map[string]any)
	if updatedSlot["dataset_id"] != "ds_b" {
		t.Fatalf("updated task dataset_id=%v, want ds_b", updatedSlot["dataset_id"])
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatal(err)
	}
}
