package main

import (
	"testing"
	"uk.ac.bris.cs/gameoflife/gol"
)

func TestPprof(t *testing.T) {
	traceParams := gol.Params{
		Turns:       1000,
		Threads:     4,
		ImageWidth:  512,
		ImageHeight: 512,
	}
	for i := 0; i < 2000; i++ {
		events := make(chan gol.Event)
		gol.Run(traceParams, events, nil)
	}
}
