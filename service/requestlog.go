package service

import (
	"einheit/boltkit/entity"
	"einheit/boltkit/util"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
)

func CreateRequestLogRoutes(router *mux.Router) {
	router.HandleFunc("/logs/list", App.ListRequestLog).Methods(http.MethodPost)
}

func (service *Service) ListRequestLog(writer http.ResponseWriter, req *http.Request) {
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

		date, ok := payload["date"].(string)
		if !ok {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("date"))
			return
		}

		email, ok := payload["email"].(string)
		if !ok {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("email"))
			return
		}

		requestType, ok := payload["requestType"].(string)
		if !ok {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("requestType"))
			return
		}

		offset, ok := payload["offset"].(float64)
		if !ok {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("offset"))
			return
		}

		requestLogs, err := entity.ListRequestLog(service.Bolt, service.Cfg.PageLimit, date, email, requestType, uint32(offset))
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		meta := map[string]interface{}{}
		meta["count"] = len(*requestLogs)
		meta["offset"] = uint32(offset)
		meta["pagesize"] = service.Cfg.PageLimit
		response := map[string]interface{}{}
		response["meta"] = meta
		response["results"] = requestLogs
		util.RespondWithJSON(writer, http.StatusOK, response)
		return
	}
}
