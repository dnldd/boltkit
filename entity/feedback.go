package entity

import (
	"einheit/boltkit/util"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/boltdb/bolt"
)

var (
	FeedbackTemplate = ""
)

// Feedback represents user feedback about the service.
type Feedback struct {
	Uuid         string `json:"uuid"`
	Details      string `json:"details"`
	User         string `json:"user"`
	Resolved     bool   `json:"resolved"`
	CreatedOn    int64  `json:"createdOn"`
	LastModified int64  `json:"lastModified"`
}

// GetFeedback fetches the feedback associated with the provided id.
func GetFeedback(id []byte, db *bolt.DB) (*Feedback, error) {
	feedback := new(Feedback)
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(util.FeedbackBucket)
		v := bucket.Get(id)
		if v == nil {
			return util.ErrKeyNotFound(string(id))
		}

		err := json.Unmarshal(v, feedback)
		return err
	})
	return feedback, err
}

// Update stores the most updated state of the feedback entity.
func (feedback *Feedback) Update(db *bolt.DB) error {
	err := db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(util.FeedbackBucket)
		feedbackBytes, err := json.Marshal(feedback)
		if err != nil {
			log.Error(util.ErrMalformedJSON)
			return util.ErrMalformedJSON
		}

		err = bucket.Put([]byte(feedback.Uuid), feedbackBytes)
		return err
	})
	return err
}

// Delete not applicable for feedback.
func (feedback *Feedback) Delete(state bool, db *bolt.DB) error {
	return util.ErrNotApplicable(reflect.TypeOf(feedback).Name())
}

// ListFeedback returns a set of feedback that match the query criteria.
func ListFeedback(db *bolt.DB, pageLimit uint32, term string, offset uint32) (*[]Feedback, error) {
	var target uint32
	feedbackList := []Feedback{}
	currFeedback := new(Feedback)
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(util.FeedbackBucket)
		cursor := bucket.Cursor()
		// Adjust data size targets based on offset and page limit
		if offset > 0 {
			target = pageLimit * (offset + 1)
		}

		if offset == 0 {
			target = pageLimit
		}

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			err := json.Unmarshal(v, currFeedback)
			if err != nil {
				return util.ErrMalformedJSON
			}

			if term != "" {
				if strings.Contains(strings.ToLower(currFeedback.User), strings.ToLower(term)) {
					feedbackList = append(feedbackList, *currFeedback)
					// Stop iterating when data target has been met.
					if uint32(len(feedbackList)) == target {
						break
					}
				}
			}

			if term == "" {
				feedbackList = append(feedbackList, *currFeedback)
				// Stop iterating when data target has been met.
				if uint32(len(feedbackList)) == target {
					break
				}
			}
		}

		// Slice the relevant data according to the page limit and offset.
		if offset > 0 {
			feedbackList = feedbackList[(pageLimit * offset):]
		}

		return nil
	})

	return &feedbackList, err
}
