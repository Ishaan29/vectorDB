package main

import (
	"fmt"

	"github.com/ishaan29/vectorDB/db"
	"github.com/ishaan29/vectorDB/storage"
)

func main() {
	// Create a new vector engine
	engine := db.NewEngine()

	// Create a new vector
	vector := storage.Vector{
		ID:        "1",
		Embedding: []float32{0.1, 0.2, 0.3},
		Metadata:  map[string]interface{}{"name": "test"},
	}

	// Insert the vector into the engine
	engine.Insert(vector)

	// Retrieve the vector from the engine
	retrievedVector, ok := engine.Get("1")
	if ok {
		fmt.Println("Retrieved Vector:", retrievedVector)
	} else {
		fmt.Println("Vector not found")
	}
}
