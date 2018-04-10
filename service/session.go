package service

import (
	"einheit/access/base58"
	"einheit/boltkit/entity"
	"einheit/boltkit/util"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/segmentio/ksuid"
	"golang.org/x/crypto/bcrypt"
)

func CreateSessionRoutes(router *mux.Router) {
	router.HandleFunc("/sessions/{id}", App.GetSession).Methods(http.MethodGet)
	router.HandleFunc("/sessions", App.CreateSession).Methods(http.MethodPost)
}

func (service *Service) GetSession(writer http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	session, err := entity.GetSession(vars["id"], App.SessionMap)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, err)
		return
	}

	util.RespondWithJSON(writer, http.StatusOK, session)
	return
}

func (service *Service) CreateSession(writer http.ResponseWriter, req *http.Request) {
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

	password, ok := payload["password"].(string)
	if !ok {
		util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("password"))
		return
	}

	// Assert the requesting user exists and the supplied password matches.
	emailB58 := base58.Encode([]byte(email))
	user, err := entity.GetUser([]byte(emailB58), service.Bolt)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, err)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, util.ErrUnauthorizedAccess)
		return
	}

	// Create the session, with expiry set to two hours from time created.
	now := time.Now()
	token := ksuid.New()
	session := entity.Session{
		User:      user.Uuid,
		Token:     token.String(),
		Access:    user.Role,
		CreatedOn: now.Unix(),
		Expiry:    util.GetFutureTime(time.Now(), 0, 2, 0, 0).Unix(),
	}
	user.LastLogin = now.Unix()
	session.Update(App.SessionMap)
	user.Update(service.Bolt)
	util.RespondWithJSON(writer, http.StatusCreated, session)
	return
}
