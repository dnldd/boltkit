package entity

import (
	"encoding/json"
	"reflect"
	"strings"

	"einheit/boltkit/util"

	"github.com/boltdb/bolt"
)

var (
	ResetTemplate = ""
)

// PassReset represents a password reset request.
type PassReset struct {
	Uuid         string `json:"uuid"`
	Email        string `json:"email"`
	User         string `json:"user"`
	ResetURL     string `json:"resetURL,omitempty"`
	Expiry       int64  `json:"expiry"`
	LastModified int64  `json:"lastModified"`
	CreatedOn    int64  `json:"createdOn"`
	Used         bool   `json:"used"`
}

// GetPassReset fetches the password reset associated with the provided id.
func GetPassReset(id []byte, db *bolt.DB) (*PassReset, error) {
	reset := new(PassReset)
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(util.PassResetBucket)
		v := bucket.Get(id)
		if v == nil {
			return util.ErrKeyNotFound(string(id))
		}

		err := json.Unmarshal(v, reset)
		return err
	})
	return reset, err
}

// Update stores the most updated state of the password reset entity.
func (reset *PassReset) Update(db *bolt.DB) error {
	err := db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(util.PassResetBucket)
		resetBytes, err := json.Marshal(reset)
		if err != nil {
			log.Error(util.ErrMalformedJSON)
			return util.ErrMalformedJSON
		}

		err = bucket.Put([]byte(reset.Uuid), resetBytes)
		return err
	})
	return err
}

// Delete not applicable for password resets.
func (reset *PassReset) Delete(state bool, db *bolt.DB) error {
	return util.ErrNotApplicable(reflect.TypeOf(reset).Name())
}

// Sanitize prepares the password reset entity to be sent a request response.
// This removes all sensitive details from the entity.
func (reset *PassReset) Sanitize() {
	reset.ResetURL = ""
}

// ListPassReset returns a set of password resets that match the query criteria.
func ListPassReset(db *bolt.DB, pageLimit uint32, term string, offset uint32) (*[]PassReset, error) {
	var target uint32
	resetList := []PassReset{}
	currReset := new(PassReset)
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(util.PassResetBucket)
		cursor := bucket.Cursor()
		// Adjust data size targets based on offset and page limit
		if offset > 0 {
			target = pageLimit * (offset + 1)
		}

		if offset == 0 {
			target = pageLimit
		}

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			err := json.Unmarshal(v, currReset)
			if err != nil {
				return util.ErrMalformedJSON
			}

			if term != "" {
				if strings.Contains(strings.ToLower(currReset.Email), strings.ToLower(term)) ||
					strings.Contains(strings.ToLower(currReset.User), strings.ToLower(term)) {
					currReset.Sanitize()
					resetList = append(resetList, *currReset)
					// Stop iterating when data target has been met.
					if uint32(len(resetList)) == target {
						break
					}
				}
			}

			if term == "" {
				resetList = append(resetList, *currReset)
				// Stop iterating when data target has been met.
				if uint32(len(resetList)) == target {
					break
				}
			}
		}

		// Slice the relevant data according to the page limit and offset.
		if offset > 0 {
			resetList = resetList[(pageLimit * offset):]
		}

		return nil
	})

	return &resetList, err
}
