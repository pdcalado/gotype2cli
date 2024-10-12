package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

//go:generate go run github.com/pdcalado/gotype2cli/cmd -type=Bar -w
type Bar struct {
	Height int `json:"height"`
}

// New creates a new Bar
func New() Bar {
	return Bar{
		Height: 12,
	}
}

// NewWithHeight creates a new Bar with the given height
func NewWithHeight(height int) Bar {
	return Bar{
		Height: 10,
	}
}

// String implements the Stringer interface
func (b *Bar) String() string {
	return fmt.Sprintf("the bar is %d meters high", +b.Height)
}

// Raise the bar by 1
func (b *Bar) Raise() {
	b.Height += 1
}

// RaiseBy raises the bar by the given amount
func (b *Bar) RaiseBy(amount int) {
	b.Height += amount
}

// RaiseFromBar raises the bar with another bar
func (b *Bar) RaiseFromBar(other *Bar) {
	b.Height += other.Height
}

// RaiseFromTwoBars raises the bar with two other bars
func (b *Bar) RaiseFromTwoBars(other1, other2 *Bar) {
	b.Height += other1.Height + other2.Height
}

// RaiseFromBars raises the bar with other bars
func (b *Bar) RaiseFromBars(others ...*Bar) {
	for _, other := range others {
		b.Height += other.Height
	}
}

// RaiseByAmountAndBars raises the bar by the given amount and other bars
func (b *Bar) RaiseByAmountAndBars(amount int, others ...*Bar) {
	b.RaiseBy(amount)
	b.RaiseFromBars(others...)
}

// Serve serves the bar in a http server
func (b *Bar) Serve(host string, port int) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(b) // nolint: errcheck
	})

	return http.ListenAndServe(fmt.Sprintf("%s:%d", host, port), nil)
}

// FetchClone fetches a bar from a http server and becomes a clone of it
func (b *Bar) FetchClone(ctx context.Context, host string, port int) error {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%s:%d", host, port), nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(b)
}

// Noop does nothing but receives context.
// Edge case testing purposes.
func (b Bar) Noop(ctx context.Context) error {
	_ = ctx
	return nil
}

func main() {
	barCmd, err := makeBarCommand()
	if err != nil {
		panic(err)
	}

	err = barCmd.Execute()
	if err != nil {
		panic(err)
	}
}
