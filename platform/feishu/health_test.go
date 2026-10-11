package feishu

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

func TestSDKConnectionHealth_UsesActualLocalWebSocketHandshakeAndDisconnect(t *testing.T) {
	var upgraded atomic.Bool
	accepted := make(chan struct{})
	closeSocket := make(chan struct{})
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/callback/ws/endpoint" {
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{
				"URL":          "ws" + strings.TrimPrefix(server.URL, "http") + "/socket?device_id=fake-device&service_id=1&ticket=fake-ticket",
				"ClientConfig": map[string]any{"PingInterval": 100, "ReconnectCount": 0, "ReconnectNonce": 0},
			}})
			return
		}
		connection, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		upgraded.Store(true)
		close(accepted)
		defer connection.Close()
		<-closeSocket
	}))
	defer server.Close()
	states := make(chan string, 4)
	options := []larkws.ClientOption{larkws.WithDomain(server.URL), larkws.WithAutoReconnect(false),
		larkws.WithLogger(&sanitizingLogger{inner: &recordingLarkLogger{}})}
	options = append(options, channelHealthOptions(func(state, _ string) { states <- state })...)
	client := larkws.NewClient("fake-app", "fake-secret", options...)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = client.Start(ctx) }()
	select {
	case <-accepted:
	case <-time.After(3 * time.Second):
		close(closeSocket)
		t.Fatal("local server did not accept the WebSocket handshake")
	}
	select {
	case state := <-states:
		if state != "online" || !upgraded.Load() {
			t.Fatalf("state=%s handshake=%v", state, upgraded.Load())
		}
	case <-time.After(3 * time.Second):
		close(closeSocket)
		t.Fatal("no health event from real SDK handshake")
	}
	close(closeSocket)
	select {
	case state := <-states:
		if state != "offline" {
			t.Fatalf("disconnect state=%s", state)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no health event from real SDK disconnect")
	}
}

func TestSDKConnectionHealth_ClassifiesTypedErrorsWithoutExposingMessages(t *testing.T) {
	if channelErrorCode(larkws.NewClientError(larkws.AuthFailed, "fake-secret")) != "credentials" {
		t.Fatal("auth error misclassified")
	}
	if channelErrorCode(larkws.NewClientError(larkws.Forbidden, "fake-secret")) != "permissions" {
		t.Fatal("permission error misclassified")
	}
	if channelErrorCode(larkws.NewServerError(500, "fake-secret")) != "network" {
		t.Fatal("server error misclassified")
	}
}
