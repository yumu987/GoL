package gol

import (
	"fmt"
	"log"
	"net/rpc"
	"os"
	"strconv"
	"time"
	stubs "uk.ac.bris.cs/gameoflife/gol-stubs"
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

func makeMatrix(height, width int) [][]uint8 {
	matrix := make([][]uint8, height)
	for i := range matrix {
		matrix[i] = make([]uint8, width)
	}
	return matrix
}

func calculateAliveCells(p Params, world [][]byte) []util.Cell {
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

func makeCall(client *rpc.Client, world [][]uint8, p Params, channel chan [][]uint8) { // p is gol.params type
	request := stubs.Request{World: world, Params: stubs.Params(p)} // Transferring world and params simultaneously
	response := new(stubs.Response)                                 // Pointer
	client.Call(stubs.GoLWorkerHandler, request, response)          // Client calls server
	channel <- response.World                                       // Transferring updated world to distributor function
}

// distributor divides the work between workers and interacts with other goroutines.
// Local controller will be responsible for IO and capturing keypresses.
// As a client on a local machine.
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.
	world := makeMatrix(p.ImageHeight, p.ImageWidth)

	final := makeMatrix(p.ImageHeight, p.ImageWidth)

	channel := make(chan [][]uint8, p.ImageHeight*p.ImageWidth) // Shared channel to transfer updated world

	// turn := 0

	// Getting data from IO, and save it to world
	c.ioCommand <- ioInput
	c.ioFilename <- strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			back := <-c.ioInput
			world[y][x] = back
		}
	}

	//server := flag.String("server", "127.0.0.1:8030", "IP:port string to connect to as server")
	//flag.Parse() // Execute the command-line parsing
	//client, _ := rpc.Dial("tcp", *server)
	//defer client.Close()

	// TODO: Make client
	server := "127.0.0.1:8030"
	client, err := rpc.Dial("tcp", server)
	if err != nil { // error happens
		log.Fatal("dialing:", err)
	}
	defer client.Close()

	done := make(chan bool) // Judge the ticker in client should be stopped or not

	// Ticker
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		for {
			select {
			case d := <-done:
				if d == true {
					ticker.Stop()
				}
			case <-ticker.C:
				request := stubs.NewRequest{B: true}                      // Transferring world and params simultaneously
				response := new(stubs.NewResponse)                        // Pointer
				client.Call(stubs.GoLNewWorkerHandler, request, response) // Client calls server
				c.events <- AliveCellsCount{response.Turn, response.Cells}
			}
		}
	}()

	// Key presses
	go func() {
		for {
			key := <-c.keyPresses
			if key == 's' {
				request := stubs.KeyRequest{Key: 's'}
				response := new(stubs.KeyResponse)
				client.Call(stubs.GoLKeyWorkerHandler, request, response)
				// err = client.Call(stubs.GoLKeyWorkerHandler, request, response)
				//if err != nil {
				//  os.Exit(10)
				//	fmt.Println(err)
				//}
				c.ioCommand <- ioOutput
				c.ioFilename <- strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(response.Turn)
				for y := 0; y < p.ImageHeight; y++ {
					for x := 0; x < p.ImageWidth; x++ {
						c.ioOutput <- response.World[y][x]
					}
				}
				c.events <- ImageOutputComplete{response.Turn, strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(response.Turn)}
			} else if key == 'q' {
				request := stubs.KeyRequest{Key: 'q'}
				response := new(stubs.KeyResponse)
				client.Call(stubs.GoLKeyWorkerHandler, request, response)
				done <- true
				os.Exit(0)
			} else if key == 'k' {
				request := stubs.KeyRequest{Key: 's'}
				response := new(stubs.KeyResponse)
				client.Call(stubs.GoLKeyWorkerHandler, request, response)
				c.ioCommand <- ioOutput
				c.ioFilename <- strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(response.Turn)
				for y := 0; y < p.ImageHeight; y++ {
					for x := 0; x < p.ImageWidth; x++ {
						c.ioOutput <- response.World[y][x]
					}
				}
				c.events <- ImageOutputComplete{response.Turn, strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(response.Turn)}
				KRequest := stubs.KeyRequest{Key: 'k'}
				KResponse := new(stubs.KeyResponse)
				client.Call(stubs.GoLKeyWorkerHandler, KRequest, KResponse)
				done <- true
				os.Exit(0)
			} else if key == 'p' {
				request := stubs.KeyRequest{Key: 'p'}
				response := new(stubs.KeyResponse)
				client.Call(stubs.GoLKeyWorkerHandler, request, response)
				c.events <- StateChange{response.Turn, Paused}
				fmt.Println("Current turn is: ", response.Turn)
				if <-c.keyPresses == 'p' {
					newRequest := stubs.BestRequest{Key: 'p'}
					newResponse := new(stubs.BestResponse)
					client.Call(stubs.GoLBestWorkerHandler, newRequest, newResponse)
					c.events <- StateChange{newResponse.Turn, Executing}
					fmt.Println("Continuing")
				}
			}
		}
	}()

	makeCall(client, world, p, channel)

	temp := <-channel // Getting updated world back and save it to temp
	world = temp      // Update 'final' world

	// 'final' is used for reporting FinalTurnComplete
	final = world

	// TODO: Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{p.Turns, calculateAliveCells(p, final)}

	// IO receives the ioOutput command and outputs the state of the board after all turns have completed
	c.ioCommand <- ioOutput
	c.ioFilename <- strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.Turns)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- final[y][x] // Sending an array of bytes to IO
		}
	}
	c.events <- ImageOutputComplete{p.Turns, strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.Turns)}

	// Final turn is completed, close ticker gracefully
	done <- true
	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
