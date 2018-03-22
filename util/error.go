package util

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var (
	// ErrUnauthorizedAccess is returned when a request does not have the
	// needed clearance to call an endpoint.
	ErrUnauthorizedAccess = errors.New("unauthorized access")

	// ErrExpiredSession is returned when a request has an expired session.
	ErrExpiredSession = errors.New("expired session")

	// ErrExpiredReset is returned when the supplied password resethas
	// already expired.
	ErrExpiredReset = errors.New("password reset has already expired")

	// ErrMalformedRequest is returned when a request has a malformed request
	// body.
	ErrMalformedRequest = errors.New("malformed request")

	// ErrMalformedPayload is returned when a request has a malformed request
	// body.
	ErrMalformedPayload = errors.New("malformed payload")

	// ErrMalformedJSON is returned when a json marshal fails for an entity.
	ErrMalformedJSON = errors.New("malformed json")

	// ErrEntityJSON is returned when an update for an entity fails.
	ErrEntityUpdate = errors.New("failed to update entity")

	// ErrReadBody is returned when the body of a request cannot be read.
	ErrReadBody = errors.New("failed to read request body")

	// ErrBcryptHash is returned when the a hash computation fails.
	ErrBcryptHash = errors.New("failed to hash user password")

	// ErrUnexpectedAuthorization is returned when a request has an unexpected
	// authorization type.
	ErrUnexpectedAuthorization = errors.New("unexpected authorization type")

	// ErrAuthorizationNotFound is returned when a request has no authorization
	// header.
	ErrAuthorizationNotFound = errors.New("authorization header not found")

	// ErrPasswordMismatch is returned when the provided password does not
	// match the stored password for a user.
	ErrPasswordMismatch = errors.New("passwords do not match")

	// ErrResetUsed is returned when the provided password reset
	// has already been used.
	ErrResetUsed = errors.New("password reset has already been used")

	// ErrNoUpdate is returned when an update call does not have updates
	// to any of the update keys of an entity.
	ErrNoUpdate = errors.New("supplied keys do not update entity")
)

// ErrKeyNotFound is returned when a query returns nothing for the key supplied.
func ErrKeyNotFound(key string) error {
	return fmt.Errorf("associated value for '%s' not found", key)
}

// ErrInvalidParam is returned when a an unexpected parameter type is
// encountered.
func ErrInvalidParameter(key string) error {
	return fmt.Errorf("invalid parameter type for '%s'", key)
}

// ErrInvalidParameterOption is returned when a an unexpected parameter option
// is encountered in a context making it invalid.
func ErrInvalidParameterOption(key string, value interface{}, expectation interface{}) error {
	return fmt.Errorf("invalid option for parameter type '%s' with value '%v', expected '%v'", key, value, expectation)
}

// ErrNotApplicable is returned when functionality is not applicable for an entity.
func ErrNotApplicable(entity string) error {
	return fmt.Errorf("functionality not applicable to entity '%s'", entity)
}

// ErrParameterGroup is returned a set of parameters are expected together but
// are not in a request.
func ErrParameterGroup(parameters []string) error {
	return fmt.Errorf("parameters '%s' are expected together in a request",
		strings.Join(parameters, string(filepath.Separator)))
}
