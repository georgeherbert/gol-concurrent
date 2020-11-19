package gol

import (
	//"fmt"
	"strconv"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFileName chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

// Sends the file name to io.go so the world can be initialised
func sendFileName(imageWidth int, imageHeight int, ioCommand chan<- ioCommand, ioFileName chan<- string) {
	ioCommand <- ioInput
	fileName := strconv.Itoa(imageWidth) + "x" + strconv.Itoa(imageHeight)
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
func calculateLiveNeighbours(neighbours []byte) int {
	liveNeighbours := 0
	for _, neighbour := range neighbours {
		if neighbour == 255 {
			liveNeighbours += 1
		}
	}
	return liveNeighbours
}

// Returns the new value of a cell given its current value and number of live neighbours
func calculateValue(item byte, liveNeighbours int) byte {
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
func calculateNextState(world [][]byte, events chan<- Event, startY int, turn int) [][]byte {
	var nextWorld [][]byte
	for y, row := range world[1:len(world) - 1] {
		nextWorld = append(nextWorld, []byte{})
		for x, element := range row {
			neighbours := getNeighbours(world, y + 1, x)
			liveNeighbours := calculateLiveNeighbours(neighbours)
			value := calculateValue(element, liveNeighbours)
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
		nextPart := calculateNextState(thePart, events, startY, turn)
		part <- nextPart
	}
}

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

// Distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	sendFileName(p.ImageWidth, p.ImageWidth, c.ioCommand, c.ioFileName)
	world := initialiseWorld(p.ImageHeight, p.ImageWidth, c.ioInput, c.events)
	parts := createPartChannels(p.Threads)
	sectionHeight := p.ImageHeight / p.Threads
	for i, part := range parts {
		startY := i * sectionHeight
		go worker(part, c.events, startY, p.Turns)
	}

	// For each turn, pass part of an image to each worker and process it, then put it back together and repeat
	var turn int
	for turn = 0; turn < p.Turns; turn++ {
		for i, part := range parts {
			startY := i * sectionHeight
			endY := startY + sectionHeight
			worldPart := getPart(world, p.Threads, i, startY, endY)
			part <- worldPart
		}
		world = [][]byte{}
		for _, part := range parts {
			world = append(world, <-part...)
		}
		c.events <- TurnComplete{
			CompletedTurns: turn,
		}
	}

	var aliveCells []util.Cell
	for y, row := range world {
		for x, element := range row {
			if element == 255 {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}
	c.events <- FinalTurnComplete{
		CompletedTurns: turn,
		Alive:          aliveCells,
	}
	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
