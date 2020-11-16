package gol

import (
	//"fmt"
	"strconv"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events chan<- Event
	ioCommand chan<- ioCommand
	ioIdle <-chan bool
	ioFileName chan<- string
	ioOutput chan<- uint8
	ioInput <-chan uint8
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.

	c.ioCommand <- ioInput

	fileName := strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageHeight)
	c.ioFileName <- fileName

	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}

	for i, row := range world {
		for j, _ := range row {
			world[i][j] = <-c.ioInput
		}
	}

	// TODO: For all initially alive cells send a CellFlipped Event.

	turn := 0

	// TODO: Execute all turns of the Game of Life.
	// TODO: Send correct Events when required, e.g. CellFlipped, TurnComplete and FinalTurnComplete.
	//		 See event.go for a list of all events.

	for y, row := range world {
		for x, element := range row {
			if element == 255 {
				//fmt.Println(x, y)
				c.events <- CellFlipped{
					CompletedTurns: turn,
					Cell: util.Cell{
						X: y,
						Y: x,
					},
				}
			}
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

	c.events <- TurnComplete {
		CompletedTurns: turn,
	}

	c.events <- FinalTurnComplete{
		CompletedTurns: turn,
		Alive: aliveCells,
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
