package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func (a *App) reportCallback(runID string, payload callbackPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/api/v1/runs/%s/callback", a.cfg.ControlHTTPBase, runID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Execution-Key", a.cfg.ControlCallbackKey)
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("callback failed with status %d", resp.StatusCode)
	}
	return nil
}
