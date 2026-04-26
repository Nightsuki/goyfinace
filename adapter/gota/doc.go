// Package gota converts goyfinace results into go-gota DataFrames.
//
// The adapter is intentionally separate from the root yfinance package so the
// core client API remains DataFrame-free for users that do not need table
// workflows.
package gota
