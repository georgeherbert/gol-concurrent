package main

import (
	"fmt"
	"os"
	"testing"
	"uk.ac.bris.cs/gameoflife/gol"
)

const (
	start = 512
	end   = 16384
)

func BenchmarkSequential(b *testing.B) {
	for size := start; size <= end; size *= 2 {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			os.Stdout = nil // Disable all program output apart from benchmark results
			params := gol.Params{
				Turns: 20,
				Threads: 1,
				ImageWidth: 64,
				ImageHeight: 64,
			}
			keyPresses := make(chan rune, 10)
			events := make(chan gol.Event, 1000)
			for i := 0; i < b.N; i++ {
				b.StartTimer()
				gol.Run(params, events, keyPresses)
				b.StopTimer()
			}
		})
	}
}

func BenchmarkParallel(b *testing.B) {
	for size := start; size <= end; size *= 2 {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			os.Stdout = nil // Disable all program output apart from benchmark results
			params := gol.Params{
				Turns: 20,
				Threads: 2,
				ImageWidth: 64,
				ImageHeight: 64,
			}
			keyPresses := make(chan rune, 10)
			events := make(chan gol.Event, 1000)
			for i := 0; i < b.N; i++ {
				b.StartTimer()
				gol.Run(params, events, keyPresses)
				b.StopTimer()
			}
		})
	}
}

// Run with "go test -bench . bench_test.go | benchgraph"
