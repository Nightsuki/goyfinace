package yfinance

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"math"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	xwebsocket "golang.org/x/net/websocket"
)

func TestDecodeStreamMessageJSON(t *testing.T) {
	msg, err := DecodeStreamMessage([]byte(`{"id":"AAPL","price":201.5,"time":1700000000,"changePercent":1.2,"volume":123}`))
	if err != nil {
		t.Fatalf("DecodeStreamMessage returned error: %v", err)
	}
	if msg.ID != "AAPL" || msg.Price != 201.5 || msg.DayVolume != 123 {
		t.Fatalf("unexpected message: %+v", msg)
	}
}

func TestDecodeStreamMessageBase64Protobuf(t *testing.T) {
	payload := appendProtoString(nil, 1, "AAPL")
	payload = appendProtoFloat32(payload, 2, 201.5)
	payload = appendProtoSInt(payload, 3, 1700000000)
	payload = appendProtoString(payload, 4, "USD")
	payload = appendProtoFloat32(payload, 8, 1.25)
	payload = appendProtoSInt(payload, 9, 12345)
	frame, _ := json.Marshal(map[string]any{"message": base64.StdEncoding.EncodeToString(payload)})

	msg, err := DecodeStreamMessage(frame)
	if err != nil {
		t.Fatalf("DecodeStreamMessage returned error: %v", err)
	}
	if msg.ID != "AAPL" || msg.Currency != "USD" || msg.Time != 1700000000 || msg.DayVolume != 12345 {
		t.Fatalf("unexpected decoded message: %+v", msg)
	}
	if math.Abs(msg.Price-201.5) > 0.001 || math.Abs(msg.ChangePercent-1.25) > 0.001 {
		t.Fatalf("unexpected floats: %+v", msg)
	}
}

func TestWebSocketSubscribeListenAndHeaders(t *testing.T) {
	received := make(chan map[string][]string, 2)
	server := httptest.NewServer(xwebsocket.Handler(func(conn *xwebsocket.Conn) {
		if got := conn.Request().UserAgent(); got != "test-agent" {
			t.Errorf("User-Agent = %q", got)
		}
		var sub map[string][]string
		if err := xwebsocket.JSON.Receive(conn, &sub); err != nil {
			t.Errorf("receive subscribe: %v", err)
			return
		}
		received <- sub
		if err := xwebsocket.Message.Send(conn, `{"id":"AAPL","price":201.5}`); err != nil {
			t.Errorf("send message: %v", err)
			return
		}
		var unsub map[string][]string
		if err := xwebsocket.JSON.Receive(conn, &unsub); err == nil {
			received <- unsub
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws := NewWebSocket(wsURL)
	ws.UserAgent = "test-agent"
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := ws.Subscribe(ctx, "aapl"); err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	gotSub := <-received
	if len(gotSub["subscribe"]) != 1 || gotSub["subscribe"][0] != "AAPL" {
		t.Fatalf("subscribe payload = %+v", gotSub)
	}

	got := make(chan StreamMessage, 1)
	errCh := make(chan error, 1)
	go func() {
		errCh <- ws.Listen(ctx, func(msg StreamMessage) {
			got <- msg
		})
	}()
	select {
	case msg := <-got:
		if msg.ID != "AAPL" || msg.Price != 201.5 {
			t.Fatalf("message = %+v", msg)
		}
	case err := <-errCh:
		t.Fatalf("Listen returned early: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for stream message")
	}

	if err := ws.Unsubscribe(context.Background(), "aapl"); err != nil {
		t.Fatalf("Unsubscribe returned error: %v", err)
	}
	gotUnsub := <-received
	if len(gotUnsub["unsubscribe"]) != 1 || gotUnsub["unsubscribe"][0] != "AAPL" {
		t.Fatalf("unsubscribe payload = %+v", gotUnsub)
	}
	cancel()
}

func TestWebSocketAutoReconnect(t *testing.T) {
	var attempts int
	server := httptest.NewServer(xwebsocket.Handler(func(conn *xwebsocket.Conn) {
		attempts++
		// First connection: drop immediately to simulate transport failure.
		if attempts == 1 {
			conn.Close()
			return
		}
		// Second connection: drain the re-subscribe payload, then send one
		// message and hold open until the test cancels.
		var sub map[string][]string
		_ = xwebsocket.JSON.Receive(conn, &sub)
		_ = xwebsocket.Message.Send(conn, `{"id":"AAPL","price":201.5}`)
		// Block until peer closes.
		var sink string
		_ = xwebsocket.Message.Receive(conn, &sink)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws := NewWebSocket(wsURL)
	ws.AutoReconnect = true
	ws.ReconnectBackoff = 5 * time.Millisecond
	ws.MaxReconnectAttempts = 3
	var reconnects int
	ws.OnReconnect = func(attempt int, err error) { reconnects++ }

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := ws.Subscribe(ctx, "aapl"); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	got := make(chan StreamMessage, 1)
	listenErr := make(chan error, 1)
	go func() {
		listenErr <- ws.Listen(ctx, func(msg StreamMessage) { got <- msg })
	}()
	select {
	case msg := <-got:
		if msg.ID != "AAPL" {
			t.Fatalf("message = %+v", msg)
		}
	case err := <-listenErr:
		t.Fatalf("Listen exited early: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for post-reconnect message")
	}
	if reconnects == 0 {
		t.Fatalf("OnReconnect never fired")
	}
	cancel()
}

func appendProtoKey(out []byte, field int, wire int) []byte {
	return binary.AppendUvarint(out, uint64(field<<3|wire))
}

func appendProtoString(out []byte, field int, value string) []byte {
	out = appendProtoKey(out, field, 2)
	out = binary.AppendUvarint(out, uint64(len(value)))
	return append(out, value...)
}

func appendProtoFloat32(out []byte, field int, value float64) []byte {
	out = appendProtoKey(out, field, 5)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], math.Float32bits(float32(value)))
	return append(out, buf[:]...)
}

func appendProtoSInt(out []byte, field int, value int64) []byte {
	out = appendProtoKey(out, field, 0)
	encoded := uint64(value<<1) ^ uint64(value>>63)
	return binary.AppendUvarint(out, encoded)
}
