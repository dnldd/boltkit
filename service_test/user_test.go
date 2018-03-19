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

// TestUser tests all user api endpoints.
func TestUser(t *testing.T) {
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

	fmt.Println("create user response: ", writer.Body.String())

	if writer.Code != http.StatusCreated {
		t.Fatalf("expected %d got %d", http.StatusOK, writer.Code)
	}

	// Get user.
	getUser := fmt.Sprint("/users/", user.Uuid)
	req, _ = http.NewRequest(http.MethodGet, getUser, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", session.Token))
	writer = httptest.NewRecorder()
	service.App.Router.ServeHTTP(writer, req)

	fmt.Println("get user response: ", writer.Body.String())

	if writer.Code != http.StatusOK {
		t.Fatalf("expected %d got %d", http.StatusOK, writer.Code)
	}

	// Update user.
	user.FirstName = "cog"

	userJSON, err := json.Marshal(user)
	if err != nil {
		t.Error(err)
	}

	updateUser := fmt.Sprint("/users/", user.Uuid)
	req, _ = http.NewRequest(http.MethodPut, updateUser, bytes.NewBuffer(userJSON))
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", session.Token))
	writer = httptest.NewRecorder()
	service.App.Router.ServeHTTP(writer, req)

	fmt.Println("update user response: ", writer.Body.String())

	if writer.Code != http.StatusOK {
		t.Fatalf("expected %d got %d", http.StatusOK, writer.Code)
	}

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

	// Reset user password.
	payload = map[string]interface{}{
		"password": "test",
		"resetId":  reset.Uuid,
	}

	payloadJSON, err = json.Marshal(payload)
	if err != nil {
		t.Error(err)
	}

	resetPassword := fmt.Sprint("/users/", user.Uuid, "/resetpassword")
	req, _ = http.NewRequest(http.MethodPut, resetPassword, bytes.NewBuffer(payloadJSON))
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", session.Token))
	writer = httptest.NewRecorder()
	service.App.Router.ServeHTTP(writer, req)

	fmt.Println("reset user password response: ", writer.Body.String())

	if writer.Code != http.StatusOK {
		t.Fatalf("expected %d got %d", http.StatusOK, writer.Code)
	}

	// Update user role.
	payload = map[string]interface{}{
		"role": util.Management,
	}

	payloadJSON, err = json.Marshal(payload)
	if err != nil {
		t.Error(err)
	}

	updateRole := fmt.Sprint("/users/", user.Uuid, "/role")
	req, _ = http.NewRequest(http.MethodPut, updateRole, bytes.NewBuffer(payloadJSON))
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", session.Token))
	writer = httptest.NewRecorder()
	service.App.Router.ServeHTTP(writer, req)

	fmt.Println("update user role response: ", writer.Body.String())

	if writer.Code != http.StatusOK {
		t.Fatalf("expected %d got %d", http.StatusOK, writer.Code)
	}

	// Delete user.
	user.Deleted = true

	userJSON, err = json.Marshal(user)
	if err != nil {
		t.Error(err)
	}

	deleteUser := fmt.Sprint("/users/", user.Uuid)
	req, _ = http.NewRequest(http.MethodDelete, deleteUser, bytes.NewBuffer(userJSON))
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", session.Token))
	writer = httptest.NewRecorder()
	service.App.Router.ServeHTTP(writer, req)

	if writer.Code != http.StatusNoContent {
		t.Fatalf("expected %d got %d", http.StatusOK, writer.Code)
	}

	// List users.
	payload = map[string]interface{}{
		"offset": 0,
		"term":   util.Management,
	}

	payloadJSON, err = json.Marshal(payload)
	if err != nil {
		t.Error(err)
	}

	req, _ = http.NewRequest(http.MethodPost, "/users/list", bytes.NewBuffer(payloadJSON))
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", session.Token))
	writer = httptest.NewRecorder()
	service.App.Router.ServeHTTP(writer, req)

	fmt.Println("list users response size: ", writer.Body.Len())

	if writer.Code != http.StatusOK {
		t.Fatalf("expected %d got %d", http.StatusOK, writer.Code)
	}
}
