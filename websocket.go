package yfinance

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	xwebsocket "golang.org/x/net/websocket"
)

// StreamMessage is one decoded Yahoo Finance streaming price update.
type StreamMessage struct {
	ID                  string
	Price               float64
	Time                int64
	Currency            string
	Exchange            string
	QuoteType           int64
	MarketHours         int64
	ChangePercent       float64
	DayVolume           int64
	DayHigh             float64
	DayLow              float64
	Change              float64
	ShortName           string
	ExpireDate          int64
	OpenPrice           float64
	PreviousClose       float64
	StrikePrice         float64
	UnderlyingSymbol    string
	OpenInterest        int64
	OptionsType         int64
	MiniOption          int64
	LastSize            int64
	Bid                 float64
	BidSize             int64
	Ask                 float64
	AskSize             int64
	PriceHint           int64
	Volume24Hr          int64
	VolumeAllCurrencies int64
	FromCurrency        string
	LastMarket          string
	CirculatingSupply   float64
	MarketCap           float64
	Raw                 map[string]any
}

// WebSocket streams live Yahoo Finance price updates.
type WebSocket struct {
	URL       string
	Origin    string
	UserAgent string

	mu            sync.Mutex
	conn          *xwebsocket.Conn
	subscriptions map[string]struct{}
}

// NewWebSocket creates a streaming client. Empty url uses Yahoo's default
// streamer endpoint.
func NewWebSocket(url string) *WebSocket {
	if url == "" {
		url = defaultStreamURL
	}
	origin := "https://finance.yahoo.com"
	if url != defaultStreamURL {
		origin = streamURLToOrigin(url)
	}
	return &WebSocket{
		URL:           url,
		Origin:        origin,
		UserAgent:     defaultUserAgent,
		subscriptions: map[string]struct{}{},
	}
}

// NewAsyncWebSocket returns a WebSocket configured for context-driven
// asynchronous Listen calls.
func NewAsyncWebSocket(url string) *WebSocket {
	return NewWebSocket(url)
}

// WebSocket creates a Yahoo Finance streaming client using this client's
// default user-agent settings.
func (c *Client) WebSocket(url string) *WebSocket {
	ws := NewWebSocket(url)
	ws.UserAgent = c.cloneWithDefaults().UserAgent
	return ws
}

// AsyncWebSocket creates a context-driven streaming client.
func (c *Client) AsyncWebSocket(url string) *WebSocket {
	return c.WebSocket(url)
}

// Connect opens the websocket connection.
func (ws *WebSocket) Connect(ctx context.Context) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if ws.conn != nil {
		return nil
	}
	config, err := xwebsocket.NewConfig(ws.URL, ws.Origin)
	if err != nil {
		return err
	}
	if ws.UserAgent == "" {
		ws.UserAgent = defaultUserAgent
	}
	config.Header.Set("User-Agent", ws.UserAgent)
	config.Dialer = &net.Dialer{}
	if deadline, ok := ctx.Deadline(); ok {
		config.Dialer.Timeout = time.Until(deadline)
	}
	conn, err := xwebsocket.DialConfig(config)
	if err != nil {
		return err
	}
	ws.conn = conn
	return nil
}

// Close closes the websocket connection.
func (ws *WebSocket) Close() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if ws.conn == nil {
		return nil
	}
	err := ws.conn.Close()
	ws.conn = nil
	return err
}

// Subscribe subscribes to one or more symbols.
func (ws *WebSocket) Subscribe(ctx context.Context, symbols ...string) error {
	normalized := normalizeSymbols(symbols)
	if len(normalized) == 0 {
		return fmt.Errorf("yfinance: no symbols")
	}
	ws.mu.Lock()
	for _, symbol := range normalized {
		ws.subscriptions[symbol] = struct{}{}
	}
	ws.mu.Unlock()
	return ws.send(ctx, map[string]any{"subscribe": normalized})
}

// Unsubscribe unsubscribes from one or more symbols.
func (ws *WebSocket) Unsubscribe(ctx context.Context, symbols ...string) error {
	normalized := normalizeSymbols(symbols)
	if len(normalized) == 0 {
		return fmt.Errorf("yfinance: no symbols")
	}
	ws.mu.Lock()
	for _, symbol := range normalized {
		delete(ws.subscriptions, symbol)
	}
	ws.mu.Unlock()
	return ws.send(ctx, map[string]any{"unsubscribe": normalized})
}

// Subscriptions returns the currently tracked subscription symbols.
func (ws *WebSocket) Subscriptions() []string {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	out := make([]string, 0, len(ws.subscriptions))
	for symbol := range ws.subscriptions {
		out = append(out, symbol)
	}
	return out
}

// Listen receives messages until the context is cancelled or Receive returns
// an error. The handler may be nil, in which case messages are simply decoded.
func (ws *WebSocket) Listen(ctx context.Context, handler func(StreamMessage)) error {
	if err := ws.Connect(ctx); err != nil {
		return err
	}
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = ws.Close()
		case <-done:
		}
	}()
	defer close(done)
	for {
		var text string
		if err := xwebsocket.Message.Receive(ws.currentConn(), &text); err != nil {
			if ctx.Err() != nil || err == io.EOF {
				return ctx.Err()
			}
			return err
		}
		msg, err := DecodeStreamMessage([]byte(text))
		if err != nil {
			return err
		}
		if handler != nil {
			handler(msg)
		}
	}
}

func (ws *WebSocket) send(ctx context.Context, payload any) error {
	if err := ws.Connect(ctx); err != nil {
		return err
	}
	return xwebsocket.JSON.Send(ws.currentConn(), payload)
}

func (ws *WebSocket) currentConn() *xwebsocket.Conn {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.conn
}

func normalizeSymbols(symbols []string) []string {
	normalized := make([]string, 0, len(symbols))
	for _, input := range symbols {
		parts := strings.FieldsFunc(input, func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
		})
		for _, symbol := range parts {
			if s := normalizeSymbol(symbol); s != "" {
				normalized = append(normalized, s)
			}
		}
	}
	return normalized
}

// DecodeStreamMessage decodes a Yahoo websocket frame. Yahoo normally sends a
// JSON envelope containing a base64 protobuf payload in the "message" field;
// plain JSON price objects are also accepted for tests and proxies.
func DecodeStreamMessage(data []byte) (StreamMessage, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return StreamMessage{}, err
	}
	if encoded, ok := raw["message"].(string); ok && encoded != "" {
		payload, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return StreamMessage{}, err
		}
		return decodePricingData(payload)
	}
	return streamMessageFromMap(raw), nil
}

func decodePricingData(payload []byte) (StreamMessage, error) {
	out := StreamMessage{Raw: map[string]any{}}
	for len(payload) > 0 {
		key, n := readUvarint(payload)
		if n <= 0 {
			return StreamMessage{}, fmt.Errorf("yfinance: invalid protobuf key")
		}
		payload = payload[n:]
		field := int(key >> 3)
		wire := int(key & 7)
		switch wire {
		case 0:
			value, n := readUvarint(payload)
			if n <= 0 {
				return StreamMessage{}, fmt.Errorf("yfinance: invalid protobuf varint")
			}
			payload = payload[n:]
			setStreamInt(&out, field, decodeStreamVarint(field, value))
		case 1:
			if len(payload) < 8 {
				return StreamMessage{}, fmt.Errorf("yfinance: invalid protobuf fixed64")
			}
			value := math.Float64frombits(binary.LittleEndian.Uint64(payload[:8]))
			payload = payload[8:]
			setStreamFloat(&out, field, value)
		case 2:
			size, n := readUvarint(payload)
			if n <= 0 || int(size) > len(payload[n:]) {
				return StreamMessage{}, fmt.Errorf("yfinance: invalid protobuf bytes")
			}
			value := string(payload[n : n+int(size)])
			payload = payload[n+int(size):]
			setStreamString(&out, field, value)
		case 5:
			if len(payload) < 4 {
				return StreamMessage{}, fmt.Errorf("yfinance: invalid protobuf fixed32")
			}
			value := float64(math.Float32frombits(binary.LittleEndian.Uint32(payload[:4])))
			payload = payload[4:]
			setStreamFloat(&out, field, value)
		default:
			return StreamMessage{}, fmt.Errorf("yfinance: unsupported protobuf wire type %d", wire)
		}
	}
	return out, nil
}

func readUvarint(data []byte) (uint64, int) {
	value, n := binary.Uvarint(data)
	return value, n
}

func decodeStreamVarint(field int, value uint64) int64 {
	switch field {
	case 3, 9, 14, 19, 21, 22, 24, 26, 27, 28, 29:
		return int64(value>>1) ^ -int64(value&1)
	default:
		return int64(value)
	}
}

func setStreamString(out *StreamMessage, field int, value string) {
	out.Raw[streamFieldName(field)] = value
	switch field {
	case 1:
		out.ID = value
	case 4:
		out.Currency = value
	case 5:
		out.Exchange = value
	case 13:
		out.ShortName = value
	case 18:
		out.UnderlyingSymbol = value
	case 30:
		out.FromCurrency = value
	case 31:
		out.LastMarket = value
	}
}

func setStreamInt(out *StreamMessage, field int, value int64) {
	out.Raw[streamFieldName(field)] = value
	switch field {
	case 3:
		out.Time = value
	case 6:
		out.QuoteType = value
	case 7:
		out.MarketHours = value
	case 9:
		out.DayVolume = value
	case 14:
		out.ExpireDate = value
	case 19:
		out.OpenInterest = value
	case 20:
		out.OptionsType = value
	case 21:
		out.MiniOption = value
	case 22:
		out.LastSize = value
	case 24:
		out.BidSize = value
	case 26:
		out.AskSize = value
	case 27:
		out.PriceHint = value
	case 28:
		out.Volume24Hr = value
	case 29:
		out.VolumeAllCurrencies = value
	}
}

func setStreamFloat(out *StreamMessage, field int, value float64) {
	out.Raw[streamFieldName(field)] = value
	switch field {
	case 2:
		out.Price = value
	case 8:
		out.ChangePercent = value
	case 10:
		out.DayHigh = value
	case 11:
		out.DayLow = value
	case 12:
		out.Change = value
	case 15:
		out.OpenPrice = value
	case 16:
		out.PreviousClose = value
	case 17:
		out.StrikePrice = value
	case 23:
		out.Bid = value
	case 25:
		out.Ask = value
	case 32:
		out.CirculatingSupply = value
	case 33:
		out.MarketCap = value
	}
}

func streamMessageFromMap(raw map[string]any) StreamMessage {
	return StreamMessage{
		ID:                stringValue(firstPresent(raw, "id", "symbol")),
		Price:             numberValue(raw["price"]),
		Time:              int64(numberValue(firstPresent(raw, "time", "timestamp"))),
		Currency:          stringValue(raw["currency"]),
		Exchange:          stringValue(raw["exchange"]),
		QuoteType:         int64(numberValue(raw["quoteType"])),
		MarketHours:       int64(numberValue(raw["marketHours"])),
		ChangePercent:     numberValue(firstPresent(raw, "changePercent", "change_percent")),
		DayVolume:         int64(numberValue(firstPresent(raw, "dayVolume", "volume"))),
		DayHigh:           numberValue(raw["dayHigh"]),
		DayLow:            numberValue(raw["dayLow"]),
		Change:            numberValue(raw["change"]),
		ShortName:         stringValue(raw["shortName"]),
		OpenPrice:         numberValue(raw["openPrice"]),
		PreviousClose:     numberValue(raw["previousClose"]),
		Bid:               numberValue(raw["bid"]),
		Ask:               numberValue(raw["ask"]),
		CirculatingSupply: numberValue(raw["circulatingSupply"]),
		MarketCap:         numberValue(firstPresent(raw, "marketcap", "marketCap")),
		Raw:               raw,
	}
}

func firstPresent(values map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			return value
		}
	}
	return nil
}

func streamFieldName(field int) string {
	names := map[int]string{
		1: "id", 2: "price", 3: "time", 4: "currency", 5: "exchange", 6: "quoteType", 7: "marketHours",
		8: "changePercent", 9: "dayVolume", 10: "dayHigh", 11: "dayLow", 12: "change", 13: "shortName",
		14: "expireDate", 15: "openPrice", 16: "previousClose", 17: "strikePrice", 18: "underlyingSymbol",
		19: "openInterest", 20: "optionsType", 21: "miniOption", 22: "lastSize", 23: "bid", 24: "bidSize",
		25: "ask", 26: "askSize", 27: "priceHint", 28: "vol_24hr", 29: "volAllCurrencies",
		30: "fromcurrency", 31: "lastMarket", 32: "circulatingSupply", 33: "marketcap",
	}
	if name, ok := names[field]; ok {
		return name
	}
	return fmt.Sprintf("field%d", field)
}

func streamURLToOrigin(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return "https://finance.yahoo.com"
	}
	if strings.EqualFold(u.Scheme, "wss") {
		return "https://" + u.Host
	}
	if strings.EqualFold(u.Scheme, "ws") {
		return "http://" + u.Host
	}
	return u.Scheme + "://" + u.Host
}
