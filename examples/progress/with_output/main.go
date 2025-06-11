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
	p.SetTotal(3)
	for i := 1; i <= 3; i++ {
		p.SetStatus(fmt.Sprintf("Working on %d/3", i))
		time.Sleep(200 * time.Millisecond)
		p.Increment(1)
	}
	p.Complete()

	output := format.OutputArray{
		Settings: settings,
		Keys:     []string{"Name"},
	}
	output.AddContents(map[string]interface{}{"Name": "Example"})
	output.Write()
}
