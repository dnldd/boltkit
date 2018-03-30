package entity

import (
	"einheit/boltkit/base58"
	"einheit/boltkit/util"
	"encoding/json"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/metakeule/fmtdate"
)

// RequestLog represents a service request log.
type RequestLog struct {
	Origin      string                 `json:"origin"`
	Requestor   string                 `json:"requestor"`
	RequestType string                 `json:"type"`
	Route       string                 `json:"route"`
	QueryParams string                 `json:"queryParams"`
	Payload     map[string]interface{} `json:"payload"`
}

// ListRequestLog returns a set of users that match the query criteria.
func ListRequestLog(db *bolt.DB, pageLimit uint32, date string, email string, requestType string, offset uint32) (*[]RequestLog, error) {
	var target uint32
	logList := []RequestLog{}
	currLog := new(RequestLog)
	time, err := fmtdate.Parse(util.TimeFormat, date)
	if err != nil {
		return nil, err
	}

	dateStr := fmtdate.Format(util.DateFormat, time)
	err = db.View(func(tx *bolt.Tx) error {
		logBucket := tx.Bucket(util.LogBucket)
		dateBucket := logBucket.Bucket([]byte(dateStr))

		token := base58.Encode([]byte(email))
		tokenBucket := dateBucket.Bucket([]byte(token))
		cursor := tokenBucket.Cursor()
		// Adjust data size targets based on offset and page limit
		if offset > 0 {
			target = pageLimit * (offset + 1)
		}

		if offset == 0 {
			target = pageLimit
		}

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			err := json.Unmarshal(v, currLog)
			if err != nil {
				return util.ErrMalformedJSON
			}

			if requestType == "" {
				logList = append(logList, *currLog)
			} else if strings.ToLower(currLog.RequestType) == strings.ToLower(requestType) {
				logList = append(logList, *currLog)
			}

			// Stop iterating when data target has been met.
			if uint32(len(logList)) == target {
				break
			}
		}

		// Slice the relevant data according to the page limit and offset.
		if offset > 0 {
			logList = logList[(pageLimit * offset):]
		}

		return nil
	})

	return &logList, err
}
