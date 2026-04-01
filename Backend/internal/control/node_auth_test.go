package control

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"araneae-go/internal/common"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestVerifyExecutorNodeKey_Success(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/verify" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get(nodeAuthHeader) != "pair-ok" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid node key"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","queue":"node-a"}`))
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()

	host, portRaw, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split host port failed: %v", err)
	}
	port, err := net.LookupPort("tcp", portRaw)
	if err != nil {
		t.Fatalf("lookup port failed: %v", err)
	}

	app := &App{}
	resp, err := app.verifyExecutorNodeKey(host, port, "pair-ok")
	if err != nil {
		t.Fatalf("verifyExecutorNodeKey failed: %v", err)
	}
	if resp.Queue != "node-a" {
		t.Fatalf("expected queue node-a, got %q", resp.Queue)
	}
}

func TestNodeAuthUnaryInterceptor_AcceptsRegisteredNodeKey(t *testing.T) {
	app := newTestControlApp(t)
	node := common.Node{
		Name:           "node-auth-test",
		Status:         "active",
		IPAddress:      "127.0.0.1",
		Port:           4280,
		GRPCPort:       9190,
		CeleryQueue:    "default",
		AuthTokenHash:  hashNodeKey("pair-ok"),
		IsEnabled:      true,
		LastActiveTime: time.Now(),
		CreatedBy:      uuid.NewString(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := app.db.Create(&node).Error; err != nil {
		t.Fatalf("create node failed: %v", err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(controlNodeAuthMetadata, "pair-ok"))
	handlerCalled := false
	_, err := app.nodeAuthUnaryInterceptor(ctx, nil, &grpc.UnaryServerInfo{}, func(context.Context, interface{}) (interface{}, error) {
		handlerCalled = true
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
	if !handlerCalled {
		t.Fatal("expected handler to be called")
	}
}

func TestNodeAuthUnaryInterceptor_RejectsUnknownNodeKey(t *testing.T) {
	app := newTestControlApp(t)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(controlNodeAuthMetadata, "invalid"))

	_, err := app.nodeAuthUnaryInterceptor(ctx, nil, &grpc.UnaryServerInfo{}, func(context.Context, interface{}) (interface{}, error) {
		return "ok", nil
	})
	if err == nil {
		t.Fatal("expected unauthenticated error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got: %v", err)
	}
	if st.Code() != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %s", st.Code())
	}
}
