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

// TestPassReset tests all password reset api endpoints.
func TestPassReset(t *testing.T) {
	err := setup()
	if err != nil {
		t.Error(err)
	}
	service.CreateSessionRoutes(service.App.Router)
	service.CreateInviteRoutes(service.App.Router)
	service.CreateUserRoutes(service.App.Router)
	service.CreatePassResetRoutes(service.App.Router)

	// Create Session.
	payload := map[string]interface{}{
		"email":    service.App.Cfg.AdminEmail,
		"password": service.App.Cfg.AdminPass,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Error(err)
	}

	req, _ := http.NewRequest(http.MethodPost, "/sessions", bytes.NewBuffer(payloadJSON))
	writer := httptest.NewRecorder()
	service.App.Router.ServeHTTP(writer, req)
	session := new(entity.Session)
	err = json.Unmarshal(writer.Body.Bytes(), session)
	if err != nil {
		t.Error(err)
	}

	defer service.App.Delete(util.SessionBucket, []byte(session.Token))

	v, err := service.App.CacheGet(util.AdminKey)
	if err != nil {
		t.Error(err)
	}

	payload = map[string]interface{}{
		"email":     "test@einheit.co",
		"role":      util.Management,
		"invitedBy": string(v),
	}

	payloadJSON, err = json.Marshal(payload)
	if err != nil {
		t.Error(err)
	}

	// Create invite.
	req, _ = http.NewRequest(http.MethodPost, "/invites", bytes.NewBuffer(payloadJSON))
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", session.Token))
	writer = httptest.NewRecorder()
	service.App.Router.ServeHTTP(writer, req)
	invite := new(entity.Invite)
	err = json.Unmarshal(writer.Body.Bytes(), invite)
	if err != nil {
		t.Error(err)
	}

	defer service.App.Delete(util.InviteBucket, []byte(invite.Uuid))

	payload = map[string]interface{}{
		"invite":    invite.Uuid,
		"firstName": "test",
		"lastName":  "user",
		"password":  "boltkit",
		"email":     "test@einheit.co",
		"role":      util.Management,
	}

	payloadJSON, err = json.Marshal(payload)
	if err != nil {
		t.Error(err)
	}

	// Create user.
	req, _ = http.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(payloadJSON))
	writer = httptest.NewRecorder()
	service.App.Router.ServeHTTP(writer, req)
	user := new(entity.User)
	err = json.Unmarshal(writer.Body.Bytes(), user)
	if err != nil {
		t.Error(err)
	}

	defer service.App.Delete(util.UserBucket, []byte(user.Uuid))

	// Create password reset.
	payload = map[string]interface{}{
		"user":  user.Uuid,
		"email": user.Email,
	}

	payloadJSON, err = json.Marshal(payload)
	if err != nil {
		t.Error(err)
	}

	req, _ = http.NewRequest(http.MethodPost, "/resets", bytes.NewBuffer(payloadJSON))
	writer = httptest.NewRecorder()
	service.App.Router.ServeHTTP(writer, req)
	reset := new(entity.PassReset)
	err = json.Unmarshal(writer.Body.Bytes(), reset)
	if err != nil {
		t.Error(err)
	}

	defer service.App.Delete(util.PassResetBucket, []byte(reset.Uuid))

	fmt.Println("create pass reset response: ", writer.Body.String())

	if writer.Code != http.StatusCreated {
		t.Fatalf("expected %d got %d", http.StatusOK, writer.Code)
	}

	// Get reset.
	getReset := fmt.Sprint("/resets/", reset.Uuid)
	req, err = http.NewRequest(http.MethodGet, getReset, nil)
	writer = httptest.NewRecorder()
	service.App.Router.ServeHTTP(writer, req)

	fmt.Println("get pass reset response: ", writer.Body.String())

	if writer.Code != http.StatusOK {
		t.Fatalf("expected %d got %d", http.StatusOK, writer.Code)
	}

	// Update reset state.
	updateReset := fmt.Sprint("/resets/", reset.Uuid)
	req, _ = http.NewRequest(http.MethodPut, updateReset, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", session.Token))
	writer = httptest.NewRecorder()
	service.App.Router.ServeHTTP(writer, req)

	fmt.Println("update pass reset response: ", writer.Body.String())

	if writer.Code != http.StatusOK {
		t.Fatalf("expected %d got %d", http.StatusOK, writer.Code)
	}
}
