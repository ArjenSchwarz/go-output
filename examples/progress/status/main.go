package main

import (
	"fmt"
	"time"

	format "github.com/ArjenSchwarz/go-output"
)

func main() {
	settings := format.NewOutputSettings()
	settings.SetOutputFormat("table")

	p := format.NewProgress(settings)
	p.SetTotal(5)
	for i := 1; i <= 5; i++ {
		p.SetStatus(fmt.Sprintf("Processing item %d/5", i))
		time.Sleep(150 * time.Millisecond)
		p.Increment(1)
	}
	p.Complete()
}
