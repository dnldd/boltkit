package entity

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/boltdb/bolt"

	"einheit/boltkit/util"
)

var (
	Pending   = "pending"
	Cancelled = "cancelled"
	Accepted  = "accepted"
)

var (
	InviteTemplate = ""
)

// Invite describes a service usage invitation.
type Invite struct {
	Uuid         string `json:"uuid"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	Status       string `json:"status"`
	LastModified int64  `json:"lastModified"`
	CreatedOn    int64  `json:"createdOn"`
	Expiry       int64  `json:"expiry"`
	InvitedBy    string `json:"invitedBy"`
	Deleted      bool   `json:"deleted"`
}

// GetInvite fetches the invite associated with the provided id.
func GetInvite(id []byte, db *bolt.DB) (*Invite, error) {
	invite := new(Invite)
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(util.InviteBucket)
		v := bucket.Get(id)
		if v == nil {
			return util.ErrKeyNotFound(string(id))
		}

		err := json.Unmarshal(v, invite)
		return err
	})
	return invite, err
}

// Update stores the most updated state of the user entity.
func (invite *Invite) Update(db *bolt.DB, mtx *sync.Mutex) error {
	mtx.Lock()
	err := db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(util.InviteBucket)
		inviteBytes, err := json.Marshal(invite)
		if err != nil {
			return util.ErrMalformedJSON
		}

		err = bucket.Put([]byte(invite.Uuid), inviteBytes)
		return err
	})
	mtx.Unlock()
	return err
}

// Delete toggles the invites entity's delete status. This determines whether
// the entity is queryable by the service, the entity will exist in storage
// regardless of state.
func (invite *Invite) Delete(state bool, db *bolt.DB, mtx *sync.Mutex) error {
	invite.Deleted = state
	invite.LastModified = time.Now().Unix()
	return invite.Update(db, mtx)
}

// ListInvites returns a set of invites that match the query criteria.
func ListInvites(db *bolt.DB, pageLimit uint32, term string, offset uint32) (*[]Invite, error) {
	var target uint32
	inviteList := []Invite{}
	currInvite := new(Invite)
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(util.InviteBucket)
		cursor := bucket.Cursor()
		// Adjust data size targets based on offset and page limit
		if offset > 0 {
			target = pageLimit * (offset + 1)
		}

		if offset == 0 {
			target = pageLimit
		}

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			err := json.Unmarshal(v, currInvite)
			if err != nil {
				return util.ErrMalformedJSON
			}

			if term != "" {
				if (strings.Contains(strings.ToLower(currInvite.Email), strings.ToLower(term)) ||
					strings.Contains(strings.ToLower(currInvite.InvitedBy), strings.ToLower(term)) ||
					strings.Contains(strings.ToLower(currInvite.Role), strings.ToLower(term))) && !currInvite.Deleted {
					inviteList = append(inviteList, *currInvite)
					// Stop iterating when data target has been met.
					if uint32(len(inviteList)) == target {
						break
					}
				}
			}

			if term == "" && !currInvite.Deleted {
				inviteList = append(inviteList, *currInvite)
				// Stop iterating when data target has been met.
				if uint32(len(inviteList)) == target {
					break
				}
			}
		}

		// Slice the relevant data according to the page limit and offset.
		if offset > 0 {
			inviteList = inviteList[(pageLimit * offset):]
		}

		return nil
	})

	return &inviteList, err
}
