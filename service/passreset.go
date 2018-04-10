package service

import (
	"einheit/boltkit/entity"
	"einheit/boltkit/util"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/segmentio/ksuid"
)

func CreatePassResetRoutes(router *mux.Router) {
	router.HandleFunc("/resets/{id}", App.GetReset).Methods(http.MethodGet)
	router.HandleFunc("/resets", App.CreateReset).Methods(http.MethodPost)
	router.HandleFunc("/resets/{id}", App.UpdateResetState).Methods(http.MethodPut)
}

func (service *Service) GetReset(writer http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	reset, err := entity.GetPassReset([]byte(vars["id"]), App.Bolt)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, err)
		return
	}

	util.RespondWithJSON(writer, http.StatusOK, reset)
	return
}

func (service *Service) CreateReset(writer http.ResponseWriter, req *http.Request) {
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

	email, ok := payload["email"].(string)
	if !ok {
		util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("email"))
		return
	}

	user, ok := payload["user"].(string)
	if !ok {
		util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("user"))
		return
	}

	// Create the password reset, setting an expiry of 5 days.
	now := time.Now()
	token := ksuid.New()
	resetURL := fmt.Sprint(service.Cfg.Frontend, "/#!/reset/", token.String())
	reset := entity.PassReset{
		User:      user,
		Email:     email,
		Uuid:      token.String(),
		CreatedOn: now.Unix(),
		Used:      false,
		ResetURL:  resetURL,
		Expiry:    util.GetFutureTime(time.Now(), 0, 5, 0, 0).Unix(),
	}

	err = reset.Update(service.Bolt)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, err)
		return
	}
	reset.Sanitize()

	// Send reset email.
	if !service.Cfg.Debug {
		template := strings.Replace(entity.ResetTemplate, "[reset]", resetURL, -1)
		template = strings.Replace(template, "[service]", service.Cfg.Server, -1)
		util.SendEmail(service.MailGun, service.Cfg.ResetEmail, service.Cfg.ResetEmail,
			"Reset your password.", template, email)
	}

	util.RespondWithJSON(writer, http.StatusCreated, reset)
	return
}

func (service *Service) UpdateResetState(writer http.ResponseWriter, req *http.Request) {
	granted, err := service.ValidateRequest([]string{util.Admin}, req)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, err)
		return
	}

	if granted {
		vars := mux.Vars(req)
		reset, err := entity.GetPassReset([]byte(vars["id"]), App.Bolt)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		reset.Used = true
		err = reset.Update(service.Bolt)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}
		reset.Sanitize()
		util.RespondWithJSON(writer, http.StatusOK, reset)
		return
	}
}
