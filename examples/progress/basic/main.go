package main

import (
	format "github.com/ArjenSchwarz/go-output"
	"time"
)

func main() {
	settings := format.NewOutputSettings()
	settings.SetOutputFormat("table")

	p := format.NewProgress(settings)
	p.SetTotal(3)
	for i := 0; i < 3; i++ {
		time.Sleep(200 * time.Millisecond)
		p.Increment(1)
	}
	p.Complete()
}
