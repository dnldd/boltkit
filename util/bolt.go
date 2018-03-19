package util

import (
	"fmt"
	"time"

	bolt "github.com/coreos/bbolt"
)

// NB: Always use a mutex lock for db writes, db reads can happen concurrently.

// OpenDB opens a bolt db, the returned db should always be closed after use,
// `defer db.Close()`.
func OpenDB(dbName string) (*bolt.DB, error) {
	db, err := bolt.Open(dbName, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open bucket: %s", err)
	}
	return db, nil
}

// CreateBucket creates a bolt bucket if it does not exist yet.
func CreateBucket(db *bolt.DB, bucketName string) error {
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return fmt.Errorf("failed to create bucket: %s", err)
		}
		return nil
	})
	return err
}
