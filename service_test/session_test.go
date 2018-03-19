package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"einheit/boltkit/entity"
	"einheit/boltkit/service"
	"einheit/boltkit/util"
)

// TestSession tests all session api endpoints.
func TestSession(t *testing.T) {
	err := setup()
	if err != nil {
		t.Error(err)
	}
	service.CreateSessionRoutes(service.App.Router)
	payload := map[string]interface{}{
		"email":    service.App.Cfg.AdminEmail,
		"password": service.App.Cfg.AdminPass,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Error(err)
	}

	// Create session.
	req, _ := http.NewRequest(http.MethodPost, "/sessions", bytes.NewBuffer(payloadJSON))
	writer := httptest.NewRecorder()
	service.App.Router.ServeHTTP(writer, req)
	fmt.Println("create session response: ", writer.Body.String())
	if writer.Code != http.StatusCreated {
		t.Fatalf("expected %d got %d", http.StatusOK, writer.Code)
	}

	session := new(entity.Session)
	err = json.Unmarshal(writer.Body.Bytes(), session)
	if err != nil {
		t.Error(err)
	}

	defer service.App.Delete(util.SessionBucket, []byte(session.Token))

	// Get Session.
	sessionGet := fmt.Sprint("/sessions/", session.Token)
	req, _ = http.NewRequest(http.MethodGet, sessionGet, nil)
	writer = httptest.NewRecorder()
	service.App.Router.ServeHTTP(writer, req)
	fmt.Println("response: ", writer.Body.String())
	if writer.Code != http.StatusOK {
		t.Fatalf("expected %d got %d", http.StatusOK, writer.Code)
	}
}
