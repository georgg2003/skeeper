package db

//go:generate TODO
type Repository interface {
	Close()
}
