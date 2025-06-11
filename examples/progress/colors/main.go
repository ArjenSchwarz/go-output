package main

import (
	"errors"
	"time"

	format "github.com/ArjenSchwarz/go-output"
)

func main() {
	settings := format.NewOutputSettings()
	settings.SetOutputFormat("table")

	success := format.NewProgress(settings)
	success.SetTotal(1)
	success.SetColor(format.ProgressColorGreen)
	time.Sleep(200 * time.Millisecond)
	success.Complete()

	failure := format.NewProgress(settings)
	failure.SetTotal(1)
	failure.SetColor(format.ProgressColorRed)
	time.Sleep(200 * time.Millisecond)
	failure.Fail(errors.New("something went wrong"))
}
