package yfinance

import (
	"context"
	"time"
)

// ActionType identifies a corporate action event.
type ActionType string

const (
	ActionDividend    ActionType = "dividend"
	ActionCapitalGain ActionType = "capitalGain"
	ActionSplit       ActionType = "split"
)

// Action is a dividend, capital-gains distribution, or stock split event.
type Action struct {
	Time  time.Time
	Type  ActionType
	Value float64
}

// Actions returns dividend, capital-gains, and split events for a symbol.
func (c *Client) Actions(ctx context.Context, symbol string, params HistoryParams) ([]Action, error) {
	history, err := c.History(ctx, symbol, actionHistoryParams(params))
	if err != nil {
		return nil, err
	}
	actions := make([]Action, 0)
	for _, candle := range history.Candles {
		if candle.Dividends != 0 {
			actions = append(actions, Action{Time: candle.Time, Type: ActionDividend, Value: candle.Dividends})
		}
		if candle.CapitalGains != 0 {
			actions = append(actions, Action{Time: candle.Time, Type: ActionCapitalGain, Value: candle.CapitalGains})
		}
		if candle.Split != 0 {
			actions = append(actions, Action{Time: candle.Time, Type: ActionSplit, Value: candle.Split})
		}
	}
	return actions, nil
}

// Actions returns dividend, capital-gains, and split events for this ticker.
func (t *Ticker) Actions(ctx context.Context, params HistoryParams) ([]Action, error) {
	return t.c().Actions(ctx, t.Symbol, params)
}

// Dividends returns dividend events for a symbol.
func (c *Client) Dividends(ctx context.Context, symbol string, params HistoryParams) ([]Action, error) {
	return c.actionsOfType(ctx, symbol, params, ActionDividend)
}

// Dividends returns dividend events for this ticker.
func (t *Ticker) Dividends(ctx context.Context, params HistoryParams) ([]Action, error) {
	return t.c().Dividends(ctx, t.Symbol, params)
}

// CapitalGains returns capital-gains distribution events for a symbol.
func (c *Client) CapitalGains(ctx context.Context, symbol string, params HistoryParams) ([]Action, error) {
	return c.actionsOfType(ctx, symbol, params, ActionCapitalGain)
}

// CapitalGains returns capital-gains distribution events for this ticker.
func (t *Ticker) CapitalGains(ctx context.Context, params HistoryParams) ([]Action, error) {
	return t.c().CapitalGains(ctx, t.Symbol, params)
}

// Splits returns stock split events for a symbol.
func (c *Client) Splits(ctx context.Context, symbol string, params HistoryParams) ([]Action, error) {
	return c.actionsOfType(ctx, symbol, params, ActionSplit)
}

// Splits returns stock split events for this ticker.
func (t *Ticker) Splits(ctx context.Context, params HistoryParams) ([]Action, error) {
	return t.c().Splits(ctx, t.Symbol, params)
}

func (c *Client) actionsOfType(ctx context.Context, symbol string, params HistoryParams, typ ActionType) ([]Action, error) {
	actions, err := c.Actions(ctx, symbol, params)
	if err != nil {
		return nil, err
	}
	filtered := actions[:0]
	for _, action := range actions {
		if action.Type == typ {
			filtered = append(filtered, action)
		}
	}
	return filtered, nil
}

func actionHistoryParams(params HistoryParams) HistoryParams {
	if params.Period == "" && params.Start.IsZero() && params.End.IsZero() {
		params.Period = PeriodMax
	}
	if params.Interval == "" {
		params.Interval = Interval1D
	}
	return params
}
