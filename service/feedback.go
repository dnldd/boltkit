package service

import (
	"einheit/boltkit/entity"
	"einheit/boltkit/util"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/segmentio/ksuid"
)

func CreateFeedbackRoutes(router *mux.Router) {
	router.HandleFunc("/feedback/{id}", App.GetFeedback).Methods(http.MethodGet)
	router.HandleFunc("/feedback", App.CreateFeedback).Methods(http.MethodPost)
	router.HandleFunc("/feedback/{id}", App.UpdateFeedbackStatus).Methods(http.MethodPut)
	router.HandleFunc("/feedback/list", App.ListFeedback).Methods(http.MethodPost)
}

func (service *Service) GetFeedback(writer http.ResponseWriter, req *http.Request) {
	granted, err := service.ValidateRequest([]string{util.Admin}, req)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, err)
		return
	}

	if granted {
		params := mux.Vars(req)
		id := params["id"]
		feedback, err := entity.GetFeedback([]byte(id), service.Bolt)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		util.RespondWithJSON(writer, http.StatusOK, feedback)
		return
	}
}

func (service *Service) CreateFeedback(writer http.ResponseWriter, req *http.Request) {
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

	user, ok := payload["user"].(string)
	if !ok {
		util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("user"))
		return
	}

	details, ok := payload["details"].(string)
	if !ok {
		util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("details"))
		return
	}

	now := time.Now()
	feedback := entity.Feedback{
		Uuid:         ksuid.New().String(),
		User:         user,
		Details:      details,
		LastModified: 0,
		CreatedOn:    now.Unix(),
	}

	err = feedback.Update(service.Bolt, service.StorageMtx)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, err)
		return
	}

	// Send feedback email.
	if !service.Cfg.Debug {
		util.SendEmail(service.MailGun, service.Cfg.FeedbackEmail, service.Cfg.FeedbackEmail,
			"Feedback submitted.", entity.FeedbackTemplate, service.Cfg.AdminEmail)
	}

	util.RespondWithJSON(writer, http.StatusCreated, feedback)
	return
}

func (service *Service) UpdateFeedbackStatus(writer http.ResponseWriter, req *http.Request) {
	granted, err := service.ValidateRequest([]string{util.Admin}, req)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, err)
		return
	}

	if granted {
		params := mux.Vars(req)
		id := params["id"]
		feedback, err := entity.GetFeedback([]byte(id), service.Bolt)
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

		resolved, _ := payload["resolved"].(bool)
		feedback.Resolved = resolved
		err = feedback.Update(service.Bolt, service.StorageMtx)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		util.RespondWithJSON(writer, http.StatusOK, feedback)
		return
	}
}

func (service *Service) ListFeedback(writer http.ResponseWriter, req *http.Request) {
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

		feedback, err := entity.ListFeedback(service.Bolt, service.Cfg.PageLimit, term, uint32(offset))
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		meta := map[string]interface{}{}
		meta["count"] = len(*feedback)
		meta["offset"] = uint32(offset)
		meta["pagesize"] = service.Cfg.PageLimit
		response := map[string]interface{}{}
		response["meta"] = meta
		response["results"] = feedback
		util.RespondWithJSON(writer, http.StatusOK, response)
		return
	}
}
