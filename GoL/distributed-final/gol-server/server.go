package main

import (
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"
	stubs "uk.ac.bris.cs/gameoflife/gol-stubs"
)

// The GoL engine will be responsible for actually processing the turns of Game of Life
// As a server on an AWS node

var mu sync.Mutex // Mutex lock to lock 'p' keypresses

var signal = make(chan bool)     // Signal to notify transfer turns and cells or not
var turnChannel = make(chan int) // Turns
var cellChannel = make(chan int) // Cells

var keySignal = make(chan bool) // Signal about key presses
var newTurn = make(chan int)    // Turns
var newWorld [][]uint8          // Global variable world

// Global variable cannot receive data from Worker (Server's function)

func makeMatrix(height, width int) [][]uint8 {
	matrix := make([][]uint8, height)
	for i := range matrix {
		matrix[i] = make([]uint8, width)
	}
	return matrix
}

func updateGol(p stubs.Params, input [][]uint8) [][]uint8 {
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

func helper(p stubs.Params, input [][]uint8) int { // Calculate the number of alive cells
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

//func calculateAliveCells(p stubs.Params, world [][]byte) []util.Cell {
//	sliceOfCells := make([]util.Cell, 0)
//	for y := 0; y < p.ImageHeight; y++ {
//		for x := 0; x < p.ImageWidth; x++ {
//			if world[y][x] == 255 {
//				newCell := util.Cell{X: x, Y: y}
//				sliceOfCells = append(sliceOfCells, newCell)
//			}
//		}
//	}
//	return sliceOfCells
//}

type GoLWorkerOperations struct{}

func (s *GoLWorkerOperations) Worker(req stubs.Request, res *stubs.Response) (err error) {
	if req.World == nil { // If the input world is nil(empty), that means this is incorrect
		err = errors.New("A world must be specified")
		return
	}

	completedTurn := 0
	temp := makeMatrix(req.Params.ImageHeight, req.Params.ImageWidth) // Temporary world
	init := makeMatrix(req.Params.ImageHeight, req.Params.ImageWidth) // Initialisation of req.World

	// State needs to be cleaned, server keeps opening, the last test goroutine remains
	// The last state will cause next test fail
	// Single-threaded test and testAlive need to be tested separately
	// For each test, the server closes and reopen again
	go func() {
		for {
			si := <-signal
			if si == true { // Transferring completedTurn and number of cells if signal is true
				turnChannel <- completedTurn
				cellChannel <- helper(req.Params, init)
			}
		}
	}()

	// Key presses
	go func() {
		for {
			si := <-keySignal
			if si == true { // Transferring completedTurn and current world if signal is true
				newTurn <- completedTurn // completedTurn
				newWorld = init          // current world
			}
		}
	}()

	if req.Params.Turns == 0 { // If the turn is initial one, which when turn = 0
		temp = req.World
		init = temp
		completedTurn = 0
		res.World = init // Report back final world
	} else { // if req.Params.Turns != 0
		init = req.World
		for i := 0; i < req.Params.Turns; i++ { // Update GoL according to rules and turns
			// TODO: Execute all turns of the Game of Life.
			mu.Lock()
			temp = updateGol(req.Params, init)
			init = temp
			completedTurn++ // Turn completed, turn = turn + 1
			mu.Unlock()
		}
		res.World = init // Turn is completed, report back final world
	}
	return
}

func (s *GoLWorkerOperations) NewWorker(req stubs.NewRequest, res *stubs.NewResponse) (err error) {
	if req.B == true {
		signal <- true
	}
	turn := <-turnChannel
	fmt.Println("Response turn: ", turn)
	res.Turn = turn
	cell := <-cellChannel
	fmt.Println("Response cell: ", cell)
	res.Cells = cell
	return
}

func (s *GoLWorkerOperations) KeyWorker(req stubs.KeyRequest, res *stubs.KeyResponse) (err error) {
	if req.Key == 's' {
		keySignal <- true
		completedTurn := <-newTurn
		res.Turn = completedTurn
		res.World = newWorld
	}
	if req.Key == 'q' { // Resetting state of server
		keySignal <- true
		completedTurn := <-newTurn
		res.Turn = completedTurn
	}
	if req.Key == 'k' {
		os.Exit(0)
	}
	if req.Key == 'p' {
		keySignal <- true
		completedTurn := <-newTurn
		res.Turn = completedTurn
		mu.Lock()
	}
	return
}

func (s *GoLWorkerOperations) BestWorker(req stubs.KeyRequest, res *stubs.KeyResponse) (err error) {
	if req.Key == 'p' {
		keySignal <- true
		completedTurn := <-newTurn
		res.Turn = completedTurn
		mu.Unlock()
	}
	return
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on") // Returns a string pointer
	flag.Parse()                                              // Execute the command-line parsing
	rand.Seed(time.Now().UnixNano())
	rpc.Register(&GoLWorkerOperations{})         // Register Update(Evolve) operation
	listener, _ := net.Listen("tcp", ":"+*pAddr) // Listen
	defer listener.Close()
	rpc.Accept(listener)
}
