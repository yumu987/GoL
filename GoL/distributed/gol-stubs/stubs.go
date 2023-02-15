package gol_stubs

var GoLWorkerHandler = "GoLWorkerOperations.Worker"
var GoLNewWorkerHandler = "GoLWorkerOperations.NewWorker"
var GoLKeyWorkerHandler = "GoLWorkerOperations.KeyWorker"
var GoLBestWorkerHandler = "GoLWorkerOperations.BestWorker"

type BestResponse struct { // Second 'p' key presses
	Turn int
}

type BestRequest struct { // Second 'p' key presses
	Key rune
}

type KeyResponse struct {
	Turn  int       // completed Turn
	World [][]uint8 // current world
}

type KeyRequest struct {
	Key rune
}

type NewResponse struct {
	Turn  int // Number of completed turn
	Cells int // Number of alive cells
}

type NewRequest struct {
	B bool // Signal to notify that transferring turn and cells back to client
}

type Response struct { // The params in Response much be upper case capital in the first letter
	World [][]uint8
}

type Request struct { // The params in Request must be upper case capital in the first letter
	World  [][]uint8
	Params Params
}

type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}
