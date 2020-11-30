package main

import (
	"fmt"
	"os"
	"testing"
	"uk.ac.bris.cs/gameoflife/gol"
)

func getThreads() []int {
	return []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
}

func BenchmarkThreads(b *testing.B) {
	for _, thread := range getThreads() {
		b.Run(fmt.Sprint(thread), func(b *testing.B) {
			os.Stdout = nil // Disable all program output apart from benchmark results
			params := gol.Params{
				Turns:       1000,
				Threads:     thread,
				ImageWidth:  512,
				ImageHeight: 512,
			}
			for i := 0; i < b.N; i++ {
				keyPresses := make(chan rune, 10)
				events := make(chan gol.Event, 1000)
				b.StartTimer()
				gol.Run(params, events, keyPresses)
				b.StopTimer()
			}
		})
	}
}

// Run with "go test -bench . bench_test.go
