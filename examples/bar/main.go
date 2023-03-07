package main

import "fmt"

//go:generate go run github.com/pdcalado/gotype2cli -type=Bar
type Bar struct {
	Height int `json:"height"`
}

func (b *Bar) String() string {
	return fmt.Sprintf("the bar is %d meters high", +b.Height)
}

func (b *Bar) Raise() {
	b.Height += 1
}

// RaiseBy raises the bar by the given amount
func (b *Bar) RaiseBy(amount int) {
	b.Height += amount
}
