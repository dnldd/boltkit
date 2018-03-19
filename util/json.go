package util

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"
)

// JSONP is a convenience type to allow for parsing JSON from a relational
// database.
type JSONP map[string]interface{}

// Value interface implementation
func (entity JSONP) Value() (driver.Value, error) {
	data, err := json.Marshal(entity)
	return data, err
}

// Scan interface implementation
func (entity *JSONP) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return errors.New("Type assertion .([]byte) failed")
	}

	var data interface{}
	err := json.Unmarshal(source, &data)
	if err != nil {
		return err
	}

	*entity, ok = data.(map[string]interface{})
	if !ok {
		return errors.New("Type assertion .(map[string]interface{}) failed")
	}

	return nil
}

// JSONList is a convenience type to allow for parsing JSON Arrays from a
// relational database.
type JSONList []interface{}

// Value interface implementation
func (entity JSONList) Value() (driver.Value, error) {
	data, err := json.Marshal(entity)
	return data, err
}

// Scan interface implementation
func (entity *JSONList) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return errors.New("Type assertion .([]byte) failed")
	}

	err := json.Unmarshal(source, entity)
	return err
}

// RespondWithError writes a JSON error message to a request.
func RespondWithError(w http.ResponseWriter, code int, err error) {
	RespondWithJSON(w, code, map[string]string{"error": err.Error()})
}

// RespondWithJSON writes a JSON payload to a request.
func RespondWithJSON(writer http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)
	writer.Write(response)
}
