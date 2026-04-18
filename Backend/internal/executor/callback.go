package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"araneae-go/internal/common"
	"araneae-go/internal/executor/contracts"
)

func (a *App) reportCallback(runID, runToken, correlationID string, payload contracts.CallbackPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/api/v1/runs/%s/callback", a.cfg.ControlHTTPBase, runID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signature := common.BuildCallbackSignature(a.cfg.ControlCallbackKey, timestamp, runID, runToken, correlationID, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Execution-Key", a.cfg.ControlCallbackKey)
	req.Header.Set("X-Run-Token", runToken)
	req.Header.Set("X-Correlation-ID", correlationID)
	req.Header.Set(common.CallbackTimestampHeader, timestamp)
	req.Header.Set(common.CallbackSignatureHeader, signature)
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
