package entity

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/boltdb/bolt"

	"einheit/boltkit/util"
)

// The User struct describes a user
type User struct {
	Uuid         string `json:"uuid"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	Password     string `json:"password,omitempty"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	LastLogin    int64  `json:"lastLogin"`
	LastModified int64  `json:"lastModified"`
	CreatedOn    int64  `json:"createdOn"`
	Deleted      bool   `json:"deleted"`
	Invite       string `json:"invite"`
}

// GetUser fetches the user associated with the provided id.
func GetUser(id []byte, db *bolt.DB) (*User, error) {
	user := new(User)
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(util.UserBucket)
		v := bucket.Get(id)
		if v == nil {
			return util.ErrKeyNotFound(string(id))
		}

		err := json.Unmarshal(v, user)
		return err
	})
	return user, err
}

// Update stores the most updated state of the user entity.
func (user *User) Update(db *bolt.DB, mtx *sync.Mutex) error {
	mtx.Lock()
	err := db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(util.UserBucket)
		userBytes, err := json.Marshal(user)
		if err != nil {
			log.Error(util.ErrMalformedJSON)
			return util.ErrMalformedJSON
		}

		err = bucket.Put([]byte(user.Uuid), userBytes)
		return err
	})
	mtx.Unlock()
	return err
}

// Delete toggles the user entity's delete status. This determines whether
// the entity is queryable by the service, the entity will exist in storage
// regardless of state.
func (user *User) Delete(state bool, db *bolt.DB, mtx *sync.Mutex) error {
	user.Deleted = state
	user.LastModified = time.Now().Unix()
	return user.Update(db, mtx)
}

// Sanitize prepares the user entity to be sent a request response.
// This removes all sensitive details from the entity.
func (user *User) Sanitize() {
	user.Password = ""
}

// ListUsers returns a set of users that match the query criteria.
func ListUsers(db *bolt.DB, pageLimit uint32, term string, offset uint32) (*[]User, error) {
	var target uint32
	userList := []User{}
	currUser := new(User)
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(util.UserBucket)
		cursor := bucket.Cursor()
		// Adjust data size targets based on offset and page limit
		if offset > 0 {
			target = pageLimit * (offset + 1)
		}

		if offset == 0 {
			target = pageLimit
		}

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			err := json.Unmarshal(v, currUser)
			if err != nil {
				return util.ErrMalformedJSON
			}

			if (strings.Contains(strings.ToLower(currUser.Email), strings.ToLower(term)) ||
				strings.Contains(strings.ToLower(currUser.Role), strings.ToLower(term))) && !currUser.Deleted {
				currUser.Sanitize()
				userList = append(userList, *currUser)
				// Stop iterating when data target has been met.
				if uint32(len(userList)) == target {
					break
				}
			}

			if term == "" && !currUser.Deleted && currUser.Role != util.Admin {
				userList = append(userList, *currUser)
				// Stop iterating when data target has been met.
				if uint32(len(userList)) == target {
					break
				}
			}
		}

		// Slice the relevant data according to the page limit and offset.
		if offset > 0 {
			userList = userList[(pageLimit * offset):]
		}

		return nil
	})

	return &userList, err
}
