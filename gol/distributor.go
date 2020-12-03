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
	for y := range world { // Create an array of bytes for each row
		world[y] = make([]byte, width)
	}
	for y, row := range world {
		for x := range row {
			cell := <-ioInput
			world[y][x] = cell // Add each cell to the row
			if cell == 255 { // If the cell is alive send a cellFlipped event
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

// Returns a slice of channels, that will each be used to communicate a section of the world between the distributor and a worker
func createPartChannels(numOfThreads int) []chan [][]byte{
	var parts []chan [][]byte
	for i := 0; i < numOfThreads; i++ {
		parts = append(parts, make(chan [][]byte))
	}
	return parts
}

// Returns a slice containing the height of each section that each worker will process
func calcSectionHeights(height int, threads int) []int {
	heightOfParts := make([]int, threads)
	for i := range heightOfParts{
		heightOfParts[i] = 0
	}
	partAssigning := 0
	for i := 0; i < height; i++ {
		heightOfParts[partAssigning] += 1
		if partAssigning == len(heightOfParts) - 1 {
			partAssigning = 0
		} else {
			partAssigning += 1
		}
	}
	return heightOfParts
}

// Returns a slice containing the initial y-values of the parts of the world that each worker will process
func calcStartYValues(sectionHeights []int) []int {
	startYValues := make([]int, len(sectionHeights))
	totalHeightAssigned := 0
	for i, height := range sectionHeights {
		startYValues[i] = totalHeightAssigned
		totalHeightAssigned += height
	}
	return startYValues
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
	for y, row := range world[1:len(world) - 1] { // Loops over each row apart from the top and bottom row
		nextWorld = append(nextWorld, []byte{})
		for x, element := range row {
			neighbours := getNeighbours(world, y + 1, x)
			liveNeighbours := calcLiveNeighbours(neighbours)
			value := calcValue(element, liveNeighbours)
			nextWorld[y] = append(nextWorld[y], value)
			if value != world[y + 1][x] { // If the value of the cell has changed send a cell flipped event
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

// Reports the number of alive cells every 2 seconds
func ticker(twoSecondTicker *time.Ticker, mutexTurnsWorld *sync.Mutex, completedTurns *int, world *[][]byte, events chan<- Event) {
	for {
		<-twoSecondTicker.C
		mutexTurnsWorld.Lock()
		events <- AliveCellsCount{
			CompletedTurns: *completedTurns,
			CellsCount:     calcNumAliveCells(*world),
		}
		mutexTurnsWorld.Unlock()
	}
}

// Receives key presses from the user and performs the appropriate action
func handleKeyPresses(keyPresses <-chan rune, mutexTurnsWorld *sync.Mutex, world *[][]byte, fileName string,
	completedTurns *int, ioCommand chan<- ioCommand, ioFileName chan<- string, ioOutput chan<- uint8,
	events chan<- Event, stop chan<- bool, pause chan<- bool) {
	paused := false
	for {
		key := <-keyPresses
		switch key {
		case 115: // Save
			mutexTurnsWorld.Lock()
			writeFile(*world, fileName, *completedTurns, ioCommand, ioFileName, ioOutput, events)
			mutexTurnsWorld.Unlock()
		case 113: // Stop
			stop <- true
		case 112: // Pause/Resume
			pause <- true
			var newState State
			if paused {
				newState = Continuing
				paused = false
			} else {
				newState = Paused
				paused = true
			}
			mutexTurnsWorld.Lock()
			events <- StateChange{*completedTurns, newState}
			mutexTurnsWorld.Unlock()
		}
	}
}

// Performs the specified number of turns of the world
func performAllTurns(turns int, stop <-chan bool, pause <-chan bool, parts []chan [][]byte, startYValues []int,
	sectionHeights []int, world *[][]byte, threads int, mutexTurnsWorld *sync.Mutex, completedTurns *int, events chan<- Event) {
	// For each turn, pass part of the board to each worker, process it, then put it back together and repeat
	turnsLoop:
		for turn := 0; turn < turns; turn++ {
			select {
			case <-stop:
				break turnsLoop
			case <-pause:
				select {
				case <-stop:
					break turnsLoop
				case <-pause:
				}
			default: // If no keys have been pressed just move onto performing the next turn of the world
			}
			for i, part := range parts { // Send the next part to each worker
				startY := startYValues[i]
				endY := startY + sectionHeights[i]
				worldPart := getPart(*world, threads, i, startY, endY)
				part <- worldPart
			}
			var nextWorld [][]byte
			for _, part := range parts { // Collect each part from each worker and build the next state of the world
				nextWorld = append(nextWorld, <-part...)
			}
			mutexTurnsWorld.Lock()
			*world = nextWorld
			*completedTurns = turn + 1 // turn + 1 because we are at the end of the turn (e.g. end of turn 0 means completed 1 turn)
			mutexTurnsWorld.Unlock()
			events <- TurnComplete{
				CompletedTurns: *completedTurns,
			}
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
	if threads == 1 { // Having 1 thread is a special case as the top and bottom row will come from the same part
		worldPart = append(worldPart, world[len(world) - 1])
		worldPart = append(worldPart, world...)
		worldPart = append(worldPart, world[0])
	} else {
		if partNum == 0 { // If it is the first part add the bottom row of the world as the top row
			worldPart = append(worldPart, world[len(world)-1])
			worldPart = append(worldPart, world[:endY + 1]...)
		} else if partNum == threads - 1 { // If it is the last part add the top row of the world as the bottom row
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
	sectionHeights := calcSectionHeights(p.ImageHeight, p.Threads)
	startYValues := calcStartYValues(sectionHeights)
	for i, part := range parts { // Starts the workers ready to receive parts to calculate the next state
		go worker(part, c.events, startYValues[i], p.Turns)
	}
	var completedTurns int
	mutexTurnsWorld := &sync.Mutex{}
	twoSecondTicker := time.NewTicker(2 * time.Second)
	go ticker(twoSecondTicker, mutexTurnsWorld, &completedTurns, &world, c.events) // Runs the ticker
	stop := make(chan bool)
	pause := make(chan bool)
	go handleKeyPresses(c.keyPresses, mutexTurnsWorld, &world, fileName, &completedTurns, c.ioCommand, c.ioFileName,
		c.ioOutput, c.events, stop, pause) // Handles key presses for the user
	performAllTurns(p.Turns, stop, pause, parts, startYValues, sectionHeights, &world, p.Threads, mutexTurnsWorld,
		&completedTurns, c.events)
	twoSecondTicker.Stop() // The ticker stops running once all turns have been performed
	mutexTurnsWorld.Lock()
	aliveCells := getAliveCells(world)
	c.events <- FinalTurnComplete{ // Send a final turn complete event to the events channel
		CompletedTurns: completedTurns,
		Alive:          aliveCells,
	}
	writeFile(world, fileName, completedTurns, c.ioCommand, c.ioFileName, c.ioOutput, c.events)
	c.ioCommand <- ioCheckIdle // Make sure that the Io has finished any output before exiting.
	<-c.ioIdle
	c.events <- StateChange{completedTurns, Quitting}
	mutexTurnsWorld.Unlock()
	close(c.events) // Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
}
