package gol

import (
	"strconv"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFileName chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	keyPresses <-chan rune
}

// Sends the file name to io.go so the world can be initialised
func sendFileName(fileName string, ioCommand chan<- ioCommand, ioFileName chan<- string) {
	ioCommand <- ioInput
	ioFileName <- fileName
}

// Returns the world with its initial values filled
func initialiseWorld(height int, width int, ioInput <-chan uint8, events chan<- Event) [][]byte {
	world := make([][]byte, height)
	for y := range world {
		world[y] = make([]byte, width)
	}
	for y, row := range world {
		for x := range row {
			cell := <-ioInput
			world[y][x] = cell
			if cell == 255 {
				events <- CellFlipped{
					CompletedTurns: 0,
					Cell: util.Cell{
						X: x,
						Y: y,
					},
				}
			}
		}
	}
	events <- TurnComplete{
		CompletedTurns: 0,
	}
	return world
}

func createPartChannels(numOfThreads int) []chan [][]byte{
	var parts []chan [][]byte
	for i := 0; i < numOfThreads; i++ {
		parts = append(parts, make(chan [][]byte))
	}
	return parts
}

// Returns the neighbours of a cell at given coordinates
func getNeighbours(world [][]byte, row int, column int) []byte {
	rowAbove, rowBelow := row - 1, row + 1
	if row == 0 {
		rowAbove = len(world[0]) - 1
	} else if row == len(world[0]) - 1 {
		rowBelow = 0
	}
	columnLeft, columnRight := column - 1, column + 1
	if column == 0 {
		columnLeft = len(world[0]) - 1
	} else if column == len(world[0]) - 1 {
		columnRight = 0
	}
	neighbours := []byte{world[rowAbove][columnLeft], world[rowAbove][column], world[rowAbove][columnRight],
		world[row][columnLeft], world[row][columnRight], world[rowBelow][columnLeft], world[rowBelow][column],
		world[rowBelow][columnRight]}
	return neighbours
}

// Returns the number of live neighbours from a set of neighbours
func calcLiveNeighbours(neighbours []byte) int {
	liveNeighbours := 0
	for _, neighbour := range neighbours {
		if neighbour == 255 {
			liveNeighbours += 1
		}
	}
	return liveNeighbours
}

// Returns the new value of a cell given its current value and number of live neighbours
func calcValue(item byte, liveNeighbours int) byte {
	calculatedValue := byte(0)
	if item == 255 {
		if liveNeighbours == 2 || liveNeighbours == 3 {
			calculatedValue = byte(255)
		}
	} else {
		if liveNeighbours == 3 {
			calculatedValue = byte(255)
		}
	}
	return calculatedValue
}

// Returns the next state of part of a world given the current state
func calcNextState(world [][]byte, events chan<- Event, startY int, turn int) [][]byte {
	var nextWorld [][]byte
	for y, row := range world[1:len(world) - 1] {
		nextWorld = append(nextWorld, []byte{})
		for x, element := range row {
			neighbours := getNeighbours(world, y + 1, x)
			liveNeighbours := calcLiveNeighbours(neighbours)
			value := calcValue(element, liveNeighbours)
			nextWorld[y] = append(nextWorld[y], value)
			if value != world[y + 1][x] {
				events <- CellFlipped{
					CompletedTurns: turn,
					Cell: util.Cell{
						X: x,
						Y: y + startY,
					},
				}
			}
		}
	}
	return nextWorld
}

// Takes part of an image, calculates the next stage, and passes it back
func worker(part chan [][]byte, events chan<- Event, startY int, turns int) {
	for turn := 0; turn < turns; turn++ {
		thePart := <-part
		nextPart := calcNextState(thePart, events, startY, turn)
		part <- nextPart
	}
}

// Returns the number of alive cells in a world
func calcNumAliveCells(world [][]byte) int {
	total := 0
	for _, row := range world {
		for _, element := range row {
			if element == 255 {
				total += 1
			}
		}
	}
	return total
}

// Returns part of a world given the number of threads, the part number, the startY, and the endY
func getPart(world [][]byte, threads int, partNum int, startY int, endY int) [][]byte {
	var worldPart [][]byte
	if threads == 1 {
		worldPart = append(worldPart, world[len(world) - 1])
		worldPart = append(worldPart, world...)
		worldPart = append(worldPart, world[0])
	} else {
		if partNum == 0 {
			worldPart = append(worldPart, world[len(world)-1])
			worldPart = append(worldPart, world[:endY + 1]...)
		} else if partNum == threads - 1 {
			worldPart = append(worldPart, world[startY - 1:]...)
			worldPart = append(worldPart, world[0])
		} else {
			worldPart = append(worldPart, world[startY - 1:endY+1]...)
		}
	}
	return worldPart
}

// Returns a slice of alive cells
func getAliveCells(world [][]byte) []util.Cell {
	var aliveCells []util.Cell
	for y, row := range world {
		for x, element := range row {
			if element == 255 {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}
	return aliveCells
}

// Writes to a file and sends the correct event
func writeFile(world [][]byte, fileName string, turns int, ioCommand chan<- ioCommand, ioFileName chan<- string,
	ioOutputChannel chan<- uint8, events chan<- Event) {
	outputFileName := fileName + "x" + strconv.Itoa(turns)
	ioCommand <- ioOutput
	ioFileName <- outputFileName
	for _, row := range world {
		for _, element := range row {
			ioOutputChannel <- element
		}
	}
	events <- ImageOutputComplete{ // implements Event
		CompletedTurns: turns,
		Filename:       outputFileName,
	}
}

// Distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	fileName := strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageHeight)
	sendFileName(fileName, c.ioCommand, c.ioFileName)
	world := initialiseWorld(p.ImageHeight, p.ImageWidth, c.ioInput, c.events)
	parts := createPartChannels(p.Threads)
	sectionHeight := p.ImageHeight / p.Threads
	for i, part := range parts {
		startY := i * sectionHeight
		go worker(part, c.events, startY, p.Turns)
	}
	var turn int
	var completedTurns int
	mutexTurnsWorld := &sync.Mutex{}
	mutexPause := &sync.Mutex{}
	mutexStop := &sync.Mutex{}
	pause := false
	var stop bool
	resume := make(chan bool)
	ticker := time.NewTicker(2 * time.Second)
	// Ticker
	go func() {
		for {
			<-ticker.C
			mutexPause.Lock()
			if pause != true {
				mutexTurnsWorld.Lock()
				c.events <- AliveCellsCount{
					CompletedTurns: completedTurns,
					CellsCount:     calcNumAliveCells(world),
				}
				mutexTurnsWorld.Unlock()
			}
			mutexPause.Unlock()
		}
	}()
	// Key presses
	go func() {
		var lastCompletedTurn int // Needed to stop the program resuming before the turn paused on has complete as would cause deadlock
		for {
			key := <-c.keyPresses
			if key == 115 { // save
				mutexTurnsWorld.Lock()
				writeFile(world, fileName, completedTurns, c.ioCommand, c.ioFileName, c.ioOutput, c.events)
				mutexTurnsWorld.Unlock()
			} else if key == 113 { // stop
				mutexPause.Lock()
				if pause != true {
					mutexStop.Lock()
					stop = true
					mutexStop.Unlock()
				}
				mutexPause.Unlock()
			} else if key == 112 { // pause/resume
				mutexPause.Lock()
				if pause == true {
					mutexTurnsWorld.Lock()
					// This loop isn't the most elegant solution, but it prevents a deadlock from occurring if pause is repeatedly pressed very quickly
					for lastCompletedTurn != completedTurns - 1 {
						mutexTurnsWorld.Unlock()
						mutexPause.Unlock()
						mutexTurnsWorld.Lock()
						mutexPause.Lock()
					}
					c.events <- StateChange{completedTurns, Continuing}
					mutexPause.Unlock() // Unlocks to let the loop for each turn to get to the block of code where it is waiting to be resumed
					resume <- true
					mutexPause.Lock()
					mutexTurnsWorld.Unlock()
				} else {
					mutexTurnsWorld.Lock()
					c.events <- StateChange{completedTurns + 1, Paused}
					lastCompletedTurn = completedTurns
					mutexTurnsWorld.Unlock()
				}
				pause = !pause
				mutexPause.Unlock()
			}
		}
	}()
	var nextWorld [][]byte
	// For each turn, pass part of the board to each worker, process it, then put it back together and repeat
	for turn = 0; turn < p.Turns; turn++ {
		mutexStop.Lock()
		if stop == true {
			mutexStop.Unlock()
			break
		} else {
			mutexStop.Unlock()
		}
		mutexPause.Lock()
		if pause == true {
			mutexPause.Unlock()
			<-resume
		} else {
			mutexPause.Unlock()
		}
		for i, part := range parts {
			startY := i * sectionHeight
			endY := startY + sectionHeight
			worldPart := getPart(world, p.Threads, i, startY, endY)
			part <- worldPart
		}
		nextWorld = [][]byte{}
		for _, part := range parts {
			nextWorld = append(nextWorld, <-part...)
		}
		mutexTurnsWorld.Lock()
		world = nextWorld
		completedTurns = turn + 1 // turn + 1 because we are at the end of the turn (e.g. end of turn 0 means completed 1 turn)
		mutexTurnsWorld.Unlock()
		c.events <- TurnComplete{
			CompletedTurns: completedTurns,
		}
	}
	ticker.Stop()
	mutexTurnsWorld.Lock()
	aliveCells := getAliveCells(world)
	c.events <- FinalTurnComplete{
		CompletedTurns: completedTurns,
		Alive:          aliveCells,
	}
	writeFile(world, fileName, turn, c.ioCommand, c.ioFileName, c.ioOutput, c.events)
	c.ioCommand <- ioCheckIdle // Make sure that the Io has finished any output before exiting.
	<-c.ioIdle
	c.events <- StateChange{completedTurns, Quitting}
	mutexTurnsWorld.Unlock()
	close(c.events) // Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
}
