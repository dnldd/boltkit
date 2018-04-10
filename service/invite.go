package service

import (
	"einheit/boltkit/entity"
	"einheit/boltkit/util"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
	"github.com/segmentio/ksuid"
)

func CreateInviteRoutes(router *mux.Router) {
	router.HandleFunc("/invites/{id}", App.GetInvite).Methods(http.MethodGet)
	router.HandleFunc("/invites", App.CreateInvite).Methods(http.MethodPost)
	router.HandleFunc("/invites/{id}", App.UpdateInvite).Methods(http.MethodPut)
	router.HandleFunc("/invites/{id}", App.DeleteInvite).Methods(http.MethodDelete)
	router.HandleFunc("/invites/list", App.ListInvites).Methods(http.MethodPost)
}

func (service *Service) GetInvite(writer http.ResponseWriter, req *http.Request) {
	granted, err := service.ValidateRequest([]string{util.Admin}, req)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, err)
		return
	}

	if granted {
		vars := mux.Vars(req)
		invite, err := entity.GetInvite([]byte(vars["id"]), service.Bolt)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		util.RespondWithJSON(writer, http.StatusOK, invite)
		return
	}
}

func (service *Service) CreateInvite(writer http.ResponseWriter, req *http.Request) {
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

		// Assert the account inviting the user is valid and has adequate
		// privileges to create an invite.
		invitedBy, ok := payload["invitedBy"].(string)
		if !ok {
			util.RespondWithError(writer, http.StatusBadRequest,
				util.ErrKeyNotFound("invitedBy"))
			return
		}

		_, err = entity.GetUser([]byte(invitedBy), service.Bolt)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest,
				util.ErrKeyNotFound("invitedBy"))
			return
		}

		currInvite := new(entity.Invite)
		match := false
		// Assert the email of the invited is not already in the system.
		err = service.Bolt.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(util.InviteBucket)
			cursor := bucket.Cursor()

			for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
				err := json.Unmarshal(v, currInvite)
				if err != nil {
					return util.ErrMalformedJSON
				}

				if currInvite.Email == email {
					match = true
					break
				}
			}

			return nil
		})

		if match {
			util.RespondWithError(writer, http.StatusBadRequest,
				errors.New("invite already exists for provided email."))
			return
		}

		now := time.Now()
		invite := entity.Invite{
			Uuid:         ksuid.New().String(),
			LastModified: 0,
			CreatedOn:    now.Unix(),
			Expiry:       util.GetFutureTime(now, 0, 7, 0, 0).Unix(),
			Deleted:      false,
			Email:        email,
			Status:       entity.Pending,
			Role:         role,
			InvitedBy:    invitedBy,
		}

		err = invite.Update(service.Bolt)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		// Send invite.
		if !service.Cfg.Debug {
			inviteURL := fmt.Sprintf(service.Cfg.Frontend, "/#!/register/", invite.Uuid)
			template := strings.Replace(entity.InviteTemplate, "[invite]", inviteURL, -1)
			template = strings.Replace(template, "[service]", service.Cfg.Server, -1)
			util.SendEmail(service.MailGun, service.Cfg.InviteEmail, service.Cfg.InviteEmail,
				"You've been invited!", template, email)
		}

		util.RespondWithJSON(writer, http.StatusCreated, invite)
		return
	}
}

func (service *Service) UpdateInvite(writer http.ResponseWriter, req *http.Request) {
	granted, err := service.ValidateRequest([]string{util.Admin}, req)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, err)
		return
	}

	if granted {
		params := mux.Vars(req)
		id := params["id"]
		invite, err := entity.GetInvite([]byte(id), service.Bolt)
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

		role, _ := payload["role"].(string)
		email, _ := payload["email"].(string)
		status, _ := payload["email"].(string)

		if role == "" && email == "" && status == "" {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrNoUpdate)
			return
		}

		now := time.Now()
		invite.LastModified = now.Unix()

		if role != "" {
			invite.Role = role
		}

		if email != "" {
			invite.Email = email
			// Resend invite.
			if !service.Cfg.Debug {
				inviteURL := fmt.Sprintf(service.Cfg.Frontend, "/#!/register/", invite.Uuid)
				template := strings.Replace(entity.InviteTemplate, "[invite]", inviteURL, -1)
				template = strings.Replace(template, "[service]", service.Cfg.Server, -1)
				util.SendEmail(service.MailGun, service.Cfg.InviteEmail, service.Cfg.InviteEmail,
					"You've been invited!", template, invite.Email)
			}
		}

		if status != "" {
			invite.Status = status
		}
		err = invite.Update(service.Bolt)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		util.RespondWithJSON(writer, http.StatusOK, invite)
		return
	}
}

func (service *Service) DeleteInvite(writer http.ResponseWriter, req *http.Request) {
	granted, err := service.ValidateRequest([]string{util.Admin}, req)
	if err != nil {
		util.RespondWithError(writer, http.StatusBadRequest, err)
		return
	}

	if granted {
		params := mux.Vars(req)
		id := params["id"]
		invite, err := entity.GetInvite([]byte(id), service.Bolt)
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

		deleted, ok := payload["deleted"].(bool)
		if !ok {
			util.RespondWithError(writer, http.StatusBadRequest, util.ErrKeyNotFound("deleted"))
			return
		}

		err = invite.Delete(deleted, service.Bolt)
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusNoContent)
		return
	}
}

func (service *Service) ListInvites(writer http.ResponseWriter, req *http.Request) {
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

		invites, err := entity.ListInvites(service.Bolt, service.Cfg.PageLimit, term, uint32(offset))
		if err != nil {
			util.RespondWithError(writer, http.StatusBadRequest, err)
			return
		}

		meta := map[string]interface{}{}
		meta["count"] = len(*invites)
		meta["offset"] = uint32(offset)
		meta["pagesize"] = service.Cfg.PageLimit
		response := map[string]interface{}{}
		response["meta"] = meta
		response["results"] = invites
		util.RespondWithJSON(writer, http.StatusOK, response)
		return
	}
}
