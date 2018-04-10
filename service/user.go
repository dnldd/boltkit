package service

import (
	"einheit/access/base58"
	"einheit/boltkit/entity"
	"einheit/boltkit/util"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

func CreateUserRoutes(router *mux.Router) {
	router.HandleFunc("/users/{id}", App.GetUser).Methods(http.MethodGet)
	router.HandleFunc("/users", App.CreateUser).Methods(http.MethodPost)
	router.HandleFunc("/users/{id}", App.UpdateUserDetails).Methods(http.MethodPut)
	router.HandleFunc("/users/{id}/resetpassword", App.ResetUserPassword).Methods(http.MethodPut)
	router.HandleFunc("/users/{id}/role", App.UpdateUserRole).Methods(http.MethodPut)
	router.HandleFunc("/users/{id}", App.DeleteUser).Methods(http.MethodDelete)
	router.HandleFunc("/users/list", App.ListUsers).Methods(http.MethodPost)
}

func (service *Service) GetUser(writer http.ResponseWriter, req *http.Request) {
	granted, err := service.ValidateRequest([]string{util.Admin}, req)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, err)
		return
	}

	if granted {
		params := mux.Vars(req)
		id := params["id"]
		user, err := entity.GetUser([]byte(id), service.Bolt)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		user.Sanitize()
		util.RespondWithJSON(writer, http.StatusOK, user)
		return
	}
}

func (service *Service) CreateUser(writer http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, util.ErrReadBody)
		return
	}
	if len(body) == 0 {
		util.RespondWithError(writer, http.StatusBadRequest, util.ErrMalformedPayload)
		return
	}

	payload := map[string]interface{}{}
	err = json.Unmarshal(body, &payload)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, util.ErrMalformedJSON)
		return
	}

	inviteRef, ok := payload["invite"].(string)
	if !ok {
		util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("invite"))
		return
	}

	firstName, ok := payload["firstName"].(string)
	if !ok {
		util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("firstName"))
		return
	}

	lastName, ok := payload["lastName"].(string)
	if !ok {
		util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("lastName"))
		return
	}

	password, ok := payload["password"].(string)
	if !ok {
		util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("password"))
		return
	}

	email, ok := payload["email"].(string)
	if !ok {
		util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("email"))
		return
	}

	role, ok := payload["role"].(string)
	if !ok {
		util.RespondWithError(writer, http.StatusBadRequest,
			util.ErrKeyNotFound("role"))
		return
	}

	invite, err := entity.GetInvite([]byte(inviteRef), service.Bolt)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, err)
	}

	if invite.Email != email {
		util.RespondWithError(writer, http.StatusBadRequest,
			errors.New("invite not associated with user being created"))
		return
	}

	if invite.Expiry < time.Now().Unix() {
		util.RespondWithError(writer, http.StatusBadRequest,
			errors.New("invite expired"))
		return
	}

	hashedPassword, err := util.BcryptHash(password)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, util.ErrBcryptHash)
		return
	}

	now := time.Now()
	user := entity.User{
		Uuid:         base58.Encode([]byte(email)),
		LastLogin:    0,
		LastModified: 0,
		CreatedOn:    now.Unix(),
		Deleted:      false,
		FirstName:    firstName,
		LastName:     lastName,
		Password:     hashedPassword,
		Email:        email,
		Role:         role,
		Invite:       inviteRef,
	}

	err = user.Update(service.Bolt)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, err)
		return
	}

	user.Sanitize()
	util.RespondWithJSON(writer, http.StatusCreated, user)
	return
}

func (service *Service) ResetUserPassword(writer http.ResponseWriter, req *http.Request) {
	granted, err := service.ValidateRequest([]string{util.Admin}, req)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, err)
		return
	}

	if granted {
		params := mux.Vars(req)
		id := params["id"]
		user, err := entity.GetUser([]byte(id), service.Bolt)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrReadBody)
			return
		}
		if len(body) == 0 {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrMalformedPayload)
			return
		}

		payload := map[string]interface{}{}
		err = json.Unmarshal(body, &payload)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrMalformedJSON)
			return
		}

		password, ok := payload["password"].(string)
		if !ok {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("password"))
			return
		}

		resetId, ok := payload["resetId"].(string)
		if !ok {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("resetId"))
			return
		}

		reset, err := entity.GetPassReset([]byte(resetId), service.Bolt)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		if reset.Used {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrResetUsed)
			return
		}

		if reset.Expiry < time.Now().Unix() {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrExpiredReset)
			return
		}

		hashedPassword, err := util.BcryptHash(password)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrBcryptHash)
			return
		}

		now := time.Now()
		user.LastModified = now.Unix()
		user.Password = hashedPassword

		err = user.Update(service.Bolt)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		user.Sanitize()
		util.RespondWithJSON(writer, http.StatusOK, user)
		return
	}
}

func (service *Service) UpdateUserRole(writer http.ResponseWriter, req *http.Request) {
	granted, err := service.ValidateRequest([]string{util.Admin}, req)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, err)
		return
	}

	if granted {
		params := mux.Vars(req)
		id := params["id"]
		user, err := entity.GetUser([]byte(id), service.Bolt)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrReadBody)
			return
		}
		if len(body) == 0 {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrMalformedPayload)
			return
		}

		payload := map[string]interface{}{}
		err = json.Unmarshal(body, &payload)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrMalformedJSON)
			return
		}

		role, ok := payload["role"].(string)
		if !ok {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("role"))
			return
		}

		now := time.Now()
		user.LastModified = now.Unix()
		user.Role = role
		err = user.Update(service.Bolt)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		user.Sanitize()
		util.RespondWithJSON(writer, http.StatusOK, user)
		return
	}
}

func (service *Service) UpdateUserDetails(writer http.ResponseWriter, req *http.Request) {
	granted, err := service.ValidateRequest([]string{util.Admin}, req)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, err)
		return
	}

	if granted {
		params := mux.Vars(req)
		id := params["id"]
		user, err := entity.GetUser([]byte(id), service.Bolt)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrReadBody)
			return
		}
		if len(body) == 0 {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrMalformedPayload)
			return
		}

		payload := map[string]interface{}{}
		err = json.Unmarshal(body, &payload)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrMalformedJSON)
			return
		}

		firstName, _ := payload["firstName"].(string)
		lastName, _ := payload["lastName"].(string)

		newPassword, newPasswordOk := payload["newPassword"].(string)
		currentPassword, currentPasswordOk := payload["currentPassword"].(string)

		if firstName == "" && lastName == "" && newPassword == "" && currentPassword == "" {
			util.RespondWithError(writer, http.StatusBadRequest,
				util.ErrNoUpdate)
			return
		}

		if (newPasswordOk && !currentPasswordOk) || (!newPasswordOk && currentPasswordOk) {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrParameterGroup([]string{"newPassword ", "currentPassword"}))
			return
		}

		if newPasswordOk && newPassword == "" {
			util.RespondWithError(writer, http.StatusBadRequest,
				util.ErrInvalidParameter("newPassword"))
			return
		}

		if newPasswordOk && currentPasswordOk {
			if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
				util.RespondWithError(writer, http.StatusBadRequest, util.ErrPasswordMismatch)
				return
			}
		}

		now := time.Now()
		user.LastModified = now.Unix()

		if firstName != "" {
			user.FirstName = firstName
		}

		if lastName != "" {
			user.LastName = lastName
		}

		if newPassword != "" {
			user.Password, err = util.BcryptHash(newPassword)
			if err != nil {
				util.RespondWithError(writer, http.StatusBadRequest, util.ErrBcryptHash)
				return
			}
		}
		err = user.Update(service.Bolt)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		user.Sanitize()
		util.RespondWithJSON(writer, http.StatusOK, user)
		return
	}
}

func (service *Service) DeleteUser(writer http.ResponseWriter, req *http.Request) {
	granted, err := service.ValidateRequest([]string{util.Admin}, req)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, err)
		return
	}

	if granted {
		params := mux.Vars(req)
		id := params["id"]
		user, err := entity.GetUser([]byte(id), service.Bolt)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrReadBody)
			return
		}
		if len(body) == 0 {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrMalformedPayload)
			return
		}

		payload := map[string]interface{}{}
		err = json.Unmarshal(body, &payload)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrMalformedJSON)
			return
		}

		deleted, _ := payload["deleted"].(bool)
		err = user.Delete(deleted, service.Bolt)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusNoContent)
		return
	}
}

func (service *Service) ListUsers(writer http.ResponseWriter, req *http.Request) {
	granted, err := service.ValidateRequest([]string{util.Admin}, req)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, err)
		return
	}

	if granted {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrReadBody)
			return
		}
		if len(body) == 0 {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrMalformedPayload)
			return
		}

		payload := map[string]interface{}{}
		err = json.Unmarshal(body, &payload)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrMalformedJSON)
			return
		}

		term, ok := payload["term"].(string)
		if !ok {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("term"))
			return
		}

		offset, ok := payload["offset"].(float64)
		if !ok {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("offset"))
			return
		}

		users, err := entity.ListUsers(service.Bolt, service.Cfg.PageLimit, term, uint32(offset))
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		meta := map[string]interface{}{}
		meta["count"] = len(*users)
		meta["offset"] = uint32(offset)
		meta["pagesize"] = service.Cfg.PageLimit
		response := map[string]interface{}{}
		response["meta"] = meta
		response["results"] = users
		util.RespondWithJSON(writer, http.StatusOK, response)
		return
	}
}
