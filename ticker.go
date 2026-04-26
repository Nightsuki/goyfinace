package yfinance

// Ticker is a symbol-scoped Yahoo Finance helper.
type Ticker struct {
	Symbol string
	client *Client
}

func (t *Ticker) c() *Client {
	return t.client.cloneWithDefaults()
}
