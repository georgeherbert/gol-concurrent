package main

import (
	"fmt"
	"os"
	"testing"
	"uk.ac.bris.cs/gameoflife/gol"
)

func theTest (b *testing.B, threads int) {
	b.Run(fmt.Sprint(threads), func(b *testing.B) {
		os.Stdout = nil // Disable all program output apart from benchmark results
		params := gol.Params{
			Turns:       1000,
			Threads:     threads,
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

func BenchmarkThreads1(b *testing.B) {
	theTest(b, 1)
}

func BenchmarkThreads2(b *testing.B) {
	theTest(b, 2)
}

func BenchmarkThreads3(b *testing.B) {
	theTest(b, 3)
}

func BenchmarkThreads4(b *testing.B) {
	theTest(b, 4)
}

func BenchmarkThreads5(b *testing.B) {
	theTest(b, 5)
}

func BenchmarkThreads6(b *testing.B) {
	theTest(b, 6)
}

func BenchmarkThreads7(b *testing.B) {
	theTest(b, 7)
}

func BenchmarkThreads8(b *testing.B) {
	theTest(b, 8)
}

func BenchmarkThreads9(b *testing.B) {
	theTest(b, 9)
}

func BenchmarkThreads10(b *testing.B) {
	theTest(b, 10)
}

func BenchmarkThreads11(b *testing.B) {
	theTest(b, 11)
}

func BenchmarkThreads12(b *testing.B) {
	theTest(b, 12)
}

func BenchmarkThreads13(b *testing.B) {
	theTest(b, 13)
}

func BenchmarkThreads14(b *testing.B) {
	theTest(b, 14)
}

func BenchmarkThreads15(b *testing.B) {
	theTest(b, 15)
}

func BenchmarkThreads16(b *testing.B) {
	theTest(b, 16)
}

// Run with "go test -bench . bench_test.go
