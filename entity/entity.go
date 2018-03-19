package entity

import "github.com/boltdb/bolt"

// Entity describes the required set of implementations (CRUD) for app entities.
type Entity interface {
	Update(db *bolt.DB) error
	Delete(db *bolt.DB) error
}
