package executor

import (
	"errors"
	"net/http"
	"testing"
)

func TestRetryableCallbackError(t *testing.T) {
	for _, status := range []int{http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusBadGateway} {
		if !retryableCallbackError(callbackStatusError{status: status}) {
			t.Fatalf("status %d should retry", status)
		}
	}
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusConflict} {
		if retryableCallbackError(callbackStatusError{status: status}) {
			t.Fatalf("status %d must not retry", status)
		}
	}
	if !retryableCallbackError(errors.New("dns timeout")) {
		t.Fatal("transport failure should retry")
	}
}
