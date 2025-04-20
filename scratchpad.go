package main

import (
	"fmt"
)

// Scratch pad for testing and experimenting with Go code.

type Vector struct {
	ID        string
	Embedding []float32
	Metadata  map[string]interface{}
}

func main() {
	fmt.Println("Hello, New Vector!")
}
