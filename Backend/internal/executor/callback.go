package executor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"araneae-go/internal/common"
	"araneae-go/internal/executor/contracts"
)

func (a *App) reportCallback(runID, runToken, correlationID string, payload contracts.CallbackPayload) error {
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		err := a.reportCallbackOnce(runID, runToken, correlationID, payload)
		if err == nil {
			return nil
		}
		lastErr = err
		if !retryableCallbackError(err) || attempt == 3 {
			break
		}
		time.Sleep(time.Duration(1<<attempt) * 250 * time.Millisecond)
	}
	return lastErr
}

type callbackStatusError struct{ status int }

func (e callbackStatusError) Error() string {
	return fmt.Sprintf("callback failed with status %d", e.status)
}

func retryableCallbackError(err error) bool {
	var status callbackStatusError
	if errors.As(err, &status) {
		return status.status == http.StatusRequestTimeout || status.status == http.StatusTooManyRequests || status.status >= 500
	}
	return err != nil // DNS, EOF and transport timeouts are transient here.
}

func (a *App) reportCallbackOnce(runID, runToken, correlationID string, payload contracts.CallbackPayload) error {
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
		return callbackStatusError{status: resp.StatusCode}
	}
	return nil
}
