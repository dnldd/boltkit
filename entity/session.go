package entity

import (
	"einheit/boltkit/util"
	"reflect"
	"sync"

	"github.com/boltdb/bolt"
	cmap "github.com/orcaman/concurrent-map"
)

// Session describes a user session.
type Session struct {
	User      string `json:"user"`
	Token     string `json:"token"`
	Access    string `json:"access"`
	CreatedOn int64  `json:"createdOn"`
	Expiry    int64  `json:"expiry"`
}

// GetSession fetches the session associated with the provided id.
func GetSession(id string, sessionMap cmap.ConcurrentMap) (*Session, error) {
	entry, ok := sessionMap.Get(id)
	if !ok {
		return nil, util.ErrKeyNotFound(id)
	}

	session := entry.(Session)
	return &session, nil
}

// Update stores the most updated state of the session entity.
func (session *Session) Update(sessionMap cmap.ConcurrentMap) {
	sessionMap.Set(session.Token, *session)
}

// Delete not applicable for sessions.
func (session *Session) Delete(state bool, db *bolt.DB, mtx *sync.Mutex) error {
	return util.ErrNotApplicable(reflect.TypeOf(session).Name())
}

// ListSessions returns a set of sessions that match the query criteria.
func ListSessions(db *bolt.DB, pageLimit uint32, term string, offset uint32) (*[]Session, error) {
	return nil, util.ErrNotApplicable("session")
}
