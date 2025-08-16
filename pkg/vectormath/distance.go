package vectormath

func CosineDistance(v1, v2 []float32) (float32, error) {
	similarity, err := CosineSimilarity(v1, v2)
	if err != nil {
		return 0, err
	}
	// Convert similarity to distance
	// Similarity range: -1 (opposite) to 1 (identical)
	// Distance range: 0 (identical) to 2 (opposite)
	return 1.0 - similarity, nil
}
