package main

import (
	"fmt"
	"os"
	"testing"
	"uk.ac.bris.cs/gameoflife/gol"
)

func getSizes() []int {
	return []int{64, 128, 256, 512}
	//return []int{16, 64, 128, 256, 512}
}

func theTest(b *testing.B, threads int, turns int) {
	for _, size := range getSizes() {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			os.Stdout = nil // Disable all program output apart from benchmark results
			params := gol.Params{
				Turns:       turns,
				Threads:     threads,
				ImageWidth:  size,
				ImageHeight: size,
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

func BenchmarkSequential(b *testing.B) {
	theTest(b, 1, 1)
}

func BenchmarkParallel8(b *testing.B) {
	theTest(b, 8, 1)
}

//func BenchmarkSequential(b *testing.B) {
//	for _, size := range getSizes() {
//		b.Run(fmt.Sprint(size), func(b *testing.B) {
//			os.Stdout = nil // Disable all program output apart from benchmark results
//			params := gol.Params{
//				Turns:       10,
//				Threads:     1,
//				ImageWidth:  size,
//				ImageHeight: size,
//			}
//			for i := 0; i < b.N; i++ {
//				keyPresses := make(chan rune, 10)
//				events := make(chan gol.Event, 1000)
//				b.StartTimer()
//				gol.Run(params, events, keyPresses)
//				b.StopTimer()
//			}
//		})
//	}
//}
//
//func BenchmarkParallel8(b *testing.B) {
//	for _, size := range getSizes() {
//		b.Run(fmt.Sprint(size), func(b *testing.B) {
//			os.Stdout = nil // Disable all program output apart from benchmark results
//			params := gol.Params{
//				Turns:       10,
//				Threads:     8,
//				ImageWidth:  size,
//				ImageHeight: size,
//			}
//			for i := 0; i < b.N; i++ {
//				keyPresses := make(chan rune, 10)
//				events := make(chan gol.Event, 1000)
//				b.StartTimer()
//				gol.Run(params, events, keyPresses)
//				b.StopTimer()
//			}
//		})
//	}
//}

// Run with "go test -bench . bench_test.go | benchgraph"
