package gol

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	keyPresses <-chan rune
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

func updateGol(p Params, input [][]uint8) [][]uint8 {
	output := makeMatrix(p.ImageHeight, p.ImageWidth)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			numberOfLiveCells := 0
			var newList []uint8
			newList = append(newList, input[(y+1+p.ImageHeight)%p.ImageHeight][(x-1+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y+1+p.ImageHeight)%p.ImageHeight][(x+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y+1+p.ImageHeight)%p.ImageHeight][(x+1+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y+p.ImageHeight)%p.ImageHeight][(x-1+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y+p.ImageHeight)%p.ImageHeight][(x+1+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y-1+p.ImageHeight)%p.ImageHeight][(x-1+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y-1+p.ImageHeight)%p.ImageHeight][(x+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y-1+p.ImageHeight)%p.ImageHeight][(x+1+p.ImageWidth)%p.ImageWidth])
			for i := range newList {
				if newList[i] == 255 {
					numberOfLiveCells++
				}
			}
			if input[y][x] == 255 && numberOfLiveCells < 2 {
				output[y][x] = 0
			}
			if input[y][x] == 255 && (numberOfLiveCells == 2 || numberOfLiveCells == 3) {
				output[y][x] = input[y][x]
			}
			if input[y][x] == 255 && numberOfLiveCells > 3 {
				output[y][x] = 0
			}
			if input[y][x] == 0 && numberOfLiveCells == 3 {
				output[y][x] = 255
			}
		}
	}
	return output
}

func work(startY, endY, startX, endX int, input [][]uint8, p Params) [][]uint8 {
	height := endY - startY
	width := endX - startX
	output := makeMatrix(height, width)
	for y := 0; y < height; y++ {
		for x := startX; x < startX+width; x++ { // for x := 0; x < width; x++
			numberOfLiveCells := 0
			var newList []uint8
			newList = append(newList, input[(y+startY+1+p.ImageHeight)%p.ImageHeight][(x-1+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y+startY+1+p.ImageHeight)%p.ImageHeight][(x+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y+startY+1+p.ImageHeight)%p.ImageHeight][(x+1+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y+startY+p.ImageHeight)%p.ImageHeight][(x-1+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y+startY+p.ImageHeight)%p.ImageHeight][(x+1+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y+startY-1+p.ImageHeight)%p.ImageHeight][(x-1+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y+startY-1+p.ImageHeight)%p.ImageHeight][(x+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y+startY-1+p.ImageHeight)%p.ImageHeight][(x+1+p.ImageWidth)%p.ImageWidth])
			for i := range newList {
				if newList[i] == 255 {
					numberOfLiveCells++
				}
			}
			if input[y+startY][x] == 255 && numberOfLiveCells < 2 {
				output[y][x] = 0
			}
			if input[y+startY][x] == 255 && (numberOfLiveCells == 2 || numberOfLiveCells == 3) {
				output[y][x] = input[y+startY][x]
			}
			if input[y+startY][x] == 255 && numberOfLiveCells > 3 {
				output[y][x] = 0
			}
			if input[y+startY][x] == 0 && numberOfLiveCells == 3 {
				output[y][x] = 255
			}
		}
	}
	return output
}

func calculateAliveCells(p Params, world [][]uint8) []util.Cell {
	sliceOfCells := make([]util.Cell, 0)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world[y][x] == 255 {
				newCell := util.Cell{X: x, Y: y}
				sliceOfCells = append(sliceOfCells, newCell)
			}
		}
	}
	return sliceOfCells
}

func worker(startY, endY, startX, endX int, data [][]uint8, out chan<- [][]uint8, p Params) {
	imagePart := work(startY, endY, startX, endX, data, p)
	out <- imagePart
}

func makeMatrix(height, width int) [][]uint8 {
	matrix := make([][]uint8, height)
	for i := range matrix {
		matrix[i] = make([]uint8, width)
	}
	return matrix
}

func helper(p Params, input [][]uint8) int {
	output := 0
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if input[y][x] == 255 {
				output++
			}
		}
	}
	return output
}

func checkFlipping(p Params, input [][]uint8) []util.Cell {
	output := util.Cell{X: 0, Y: 0}
	sliceOfCells := make([]util.Cell, 0)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			var newList []uint8
			var numberOfLiveCells int
			newList = append(newList, input[(y+1+p.ImageHeight)%p.ImageHeight][(x-1+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y+1+p.ImageHeight)%p.ImageHeight][(x+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y+1+p.ImageHeight)%p.ImageHeight][(x+1+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y+p.ImageHeight)%p.ImageHeight][(x-1+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y+p.ImageHeight)%p.ImageHeight][(x+1+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y-1+p.ImageHeight)%p.ImageHeight][(x-1+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y-1+p.ImageHeight)%p.ImageHeight][(x+p.ImageWidth)%p.ImageWidth])
			newList = append(newList, input[(y-1+p.ImageHeight)%p.ImageHeight][(x+1+p.ImageWidth)%p.ImageWidth])
			for i := range newList {
				if newList[i] == 255 {
					numberOfLiveCells++
				}
			}
			if input[y][x] == 255 && numberOfLiveCells < 2 {
				output = util.Cell{X: x, Y: y}
				sliceOfCells = append(sliceOfCells, output)
			} else if input[y][x] == 255 && numberOfLiveCells > 3 {
				output = util.Cell{X: x, Y: y}
				sliceOfCells = append(sliceOfCells, output)
			} else if input[y][x] == 0 && numberOfLiveCells == 3 {
				output = util.Cell{X: x, Y: y}
				sliceOfCells = append(sliceOfCells, output)
			}
		}
	}
	return sliceOfCells
}

//func flipping(p Params, old [][]uint8, new [][]uint8) []util.Cell { // Compare old world with new world to check which cell can be evolved
//	output := util.Cell{X: 0, Y: 0} // Initialisation of cell
//	sliceOfCells := make([]util.Cell, 0)
//	for y := 0; y < p.ImageHeight; y++ {
//		for x := 0; x < p.ImageWidth; x++ {
//			if old[y][x] != new[y][x] {
//				output = util.Cell{X: x, Y: y}
//				sliceOfCells = append(sliceOfCells, output)
//			}
//		}
//	}
//	return sliceOfCells
//}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.
	world := makeMatrix(p.ImageHeight, p.ImageWidth) // [y, x] / (y, x)

	final := makeMatrix(p.ImageHeight, p.ImageWidth) // Final world [y, x] / (y, x)

	completedTurn := 0 // completedTurn will plus 1 to report alive cells count after every turn finished

	var newPixelData [][]uint8 // Complete nil empty world

	var mu sync.Mutex

	// turn := 0
	turn := p.Turns

	// For reading in the initial PGM image
	c.ioCommand <- ioInput
	c.ioFilename <- strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth)

	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			back := <-c.ioInput
			world[y][x] = back
		}
	}

	// Before processing(the program starts), after distributor got the data from IO, report all alive cells at first
	f := calculateAliveCells(p, world)
	for _, v := range f {
		c.events <- CellFlipped{0, v}
	}

	// If done == true, it means that the final turn is complete and stop the ticker immediately
	done := make(chan bool)

	// Ticker
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		for {
			select {
			case d := <-done:
				if d == true {
					ticker.Stop() // Close ticker immediately
				}
			case <-ticker.C:
				c.events <- AliveCellsCount{completedTurn, helper(p, world)}
			}
		}
	}()

	// key presses
	go func() {
		for {
			key := <-c.keyPresses
			if key == 's' { // Generate a PGM file with the current state of the board
				c.ioCommand <- ioOutput
				c.ioFilename <- strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(completedTurn)
				for y := 0; y < p.ImageHeight; y++ {
					for x := 0; x < p.ImageWidth; x++ {
						c.ioOutput <- world[y][x]
					}
				}
				c.events <- ImageOutputComplete{completedTurn, strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(completedTurn)}
			} else if key == 'q' { // Generate a PGM file with the current state of the board and then terminate the program
				c.ioCommand <- ioOutput
				c.ioFilename <- strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(completedTurn)
				for y := 0; y < p.ImageHeight; y++ {
					for x := 0; x < p.ImageWidth; x++ {
						c.ioOutput <- world[y][x]
					}
				}
				c.events <- ImageOutputComplete{completedTurn, strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(completedTurn)}
				done <- true
				c.ioCommand <- ioCheckIdle
				<-c.ioIdle
				c.events <- StateChange{completedTurn, Quitting}
				close(c.events)
				os.Exit(0)
			} else if key == 'p' { // Pause the processing and print the current turn that is being processed
				// While the execution is paused, q and s should not work
				c.events <- StateChange{completedTurn, Paused}
				fmt.Println("Current turn is: ", completedTurn)
				mu.Lock()
				if <-c.keyPresses == 'p' { // If p is pressed again resume the processing and print "Continuing"
					c.events <- StateChange{completedTurn, Executing}
					fmt.Println("Continuing")
					mu.Unlock()
				}
			}
		}
	}()

	if p.Threads == 1 {
		// TODO: Execute all turns of the Game of Life.
		for t := 0; t < p.Turns; t++ {
			temp := makeMatrix(p.ImageHeight, p.ImageWidth)
			o := checkFlipping(p, world)
			for _, v := range o {
				c.events <- CellFlipped{t + 1, v}
			}
			temp = updateGol(p, world)
			//o := flipping(p, world, temp) // Compare old world and new world before world gets updated
			//for _, v := range o {
			//	c.events <- CellFlipped{t + 1, v}
			//}
			world = temp // Each round, it covers the original one and takes the data for next round of evolving
			c.events <- TurnComplete{t + 1}
			completedTurn++ // completedTurn = completedTurn + 1
		}
		final = world
	} else { // Other threads
		workerHeight := p.ImageHeight / p.Threads

		out := make([]chan [][]byte, p.Threads)
		for i := range out {
			out[i] = make(chan [][]byte)
		}

		// TODO: Execute all turns of the Game of Life.
		for i := 0; i < p.Turns; i++ {
			newPixelData = make([][]uint8, 0) // newPixelData = makeMatrix(0, 0)
			o := checkFlipping(p, world)
			for _, v := range o {
				c.events <- CellFlipped{i + 1, v}
			}
			if p.ImageHeight%p.Threads == 0 { // Can be fully divisible
				for j := 0; j < p.Threads; j++ {
					go worker(j*workerHeight, (j+1)*workerHeight, 0, p.ImageWidth, world, out[j], p)
				}
				mu.Lock() // Actually, we can also lock it during world gets updated
				for a := 0; a < p.Threads; a++ {
					part := <-out[a]
					newPixelData = append(newPixelData, part...)
				}
				mu.Unlock()
			} else { // Cannot be fully divisible (p.ImageHeight%p.Threads != 0)
				for j := 0; j < p.Threads; j++ {
					if j == (p.Threads - 1) { // Final turn, let the final worker do more things to finish this graph
						go worker(j*workerHeight, p.ImageHeight, 0, p.ImageWidth, world, out[j], p)
					} else {
						go worker(j*workerHeight, (j+1)*workerHeight, 0, p.ImageWidth, world, out[j], p)
					}
				}
				mu.Lock() // Actually, we can also lock it during world gets updated
				for a := 0; a < p.Threads; a++ {
					part := <-out[a]
					newPixelData = append(newPixelData, part...)
				}
				mu.Unlock()
			}
			//o := flipping(p, world, newPixelData) // Compare old world and new world before world gets updated
			//for _, v := range o {
			//	c.events <- CellFlipped{i + 1, v}
			//}
			//mu.Lock() // Testing stuck of mutex lock
			world = newPixelData // Update world, it covers the original one and takes data for the next round of evolving
			c.events <- TurnComplete{i + 1}
			completedTurn++ // completedTurn = completedTurn + 1
			//mu.Unlock()
		}
		final = world
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{turn, calculateAliveCells(p, final)}

	// IO receives the ioOutput command and outputs the state of the board after all turns have completed
	c.ioCommand <- ioOutput
	c.ioFilename <- strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.Turns)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- final[y][x] // Sending an array of bytes to IO
		}
	}
	c.events <- ImageOutputComplete{turn, strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.Turns)}

	// The final turn is completed, report true to done in order to stop the ticker
	done <- true

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
