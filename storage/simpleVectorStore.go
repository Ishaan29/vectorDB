package storage

type SimpleVectorStore interface {
	Insert(vector Vector)
	Get(id string) (Vector, bool)
}
