package main

import "fmt"

//go:generate go run github.com/pdcalado/gotype2cli -type=Bar
type Bar struct {
	Height int `json:"height"`
}

// New creates a new Bar
func New() Bar {
	return Bar{
		Height: 12,
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
