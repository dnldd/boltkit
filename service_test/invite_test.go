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

// TestInvite tests all invite api endpoints.
func TestInvite(t *testing.T) {
	err := setup()
	if err != nil {
		t.Error(err)
	}
	service.CreateSessionRoutes(service.App.Router)
	service.CreateInviteRoutes(service.App.Router)

	// Create Session
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

	fmt.Println("create invite response: ", writer.Body.String())

	if writer.Code != http.StatusCreated {
		t.Fatalf("expected %d got %d", http.StatusOK, writer.Code)
	}

	// Get invite.
	invite := new(entity.Invite)
	err = json.Unmarshal(writer.Body.Bytes(), invite)
	if err != nil {
		t.Error(err)
	}

	defer service.App.Delete(util.InviteBucket, []byte(invite.Uuid))

	getInvite := fmt.Sprint("/invites/", invite.Uuid)
	req, _ = http.NewRequest(http.MethodGet, getInvite, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", session.Token))
	writer = httptest.NewRecorder()
	service.App.Router.ServeHTTP(writer, req)

	fmt.Println("get invite response: ", writer.Body.String())

	if writer.Code != http.StatusOK {
		t.Fatalf("expected %d got %d", http.StatusOK, writer.Code)
	}

	// Update invite.
	invite.Role = util.Management

	inviteJSON, err := json.Marshal(invite)
	if err != nil {
		t.Error(err)
	}

	updateInvite := fmt.Sprint("/invites/", invite.Uuid)
	req, _ = http.NewRequest(http.MethodPut, updateInvite, bytes.NewBuffer(inviteJSON))
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", session.Token))
	writer = httptest.NewRecorder()
	service.App.Router.ServeHTTP(writer, req)

	fmt.Println("update invite response: ", writer.Body.String())

	if writer.Code != http.StatusOK {
		t.Fatalf("expected %d got %d", http.StatusOK, writer.Code)
	}

	// Delete invite.
	invite.Deleted = true

	inviteJSON, err = json.Marshal(invite)
	if err != nil {
		t.Error(err)
	}

	deleteInvite := fmt.Sprint("/invites/", invite.Uuid)
	req, _ = http.NewRequest(http.MethodDelete, deleteInvite, bytes.NewBuffer(inviteJSON))
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", session.Token))
	writer = httptest.NewRecorder()
	service.App.Router.ServeHTTP(writer, req)

	if writer.Code != http.StatusNoContent {
		t.Fatalf("expected %d got %d", http.StatusOK, writer.Code)
	}

	// List invites.
	payload = map[string]interface{}{
		"offset": 0,
		"term":   "test@einheit.co",
	}

	payloadJSON, err = json.Marshal(payload)
	if err != nil {
		t.Error(err)
	}

	req, _ = http.NewRequest(http.MethodPost, "/invites/list", bytes.NewBuffer(payloadJSON))
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", session.Token))
	writer = httptest.NewRecorder()
	service.App.Router.ServeHTTP(writer, req)

	fmt.Println("list invite response size: ", writer.Body.Len())

	if writer.Code != http.StatusOK {
		t.Fatalf("expected %d got %d", http.StatusOK, writer.Code)
	}
}
