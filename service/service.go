package service

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
	mailgun "github.com/mailgun/mailgun-go"
	"github.com/metakeule/fmtdate"
	cmap "github.com/orcaman/concurrent-map"

	"einheit/boltkit/base58"
	"einheit/boltkit/entity"
	"einheit/boltkit/util"
)

var (
	// The application.
	App *Service
	// The http server.
	Server http.Server
)

// Service represents the application.
type Service struct {
	Bolt       *bolt.DB
	Cfg        *util.Config
	SessionMap cmap.ConcurrentMap
	MailGun    mailgun.Mailgun
	HTTPClient *http.Client
	StorageMtx *sync.Mutex
	Router     *mux.Router
	S3         *util.S3Connection
}

// NewService initialises the service object. It also establishes all
// component connections.
func NewService(configPath string) (*Service, error) {
	service := new(Service)
	var err error

	// Load the configuration.
	service.Cfg, err = util.NewConfig(configPath)
	if err != nil {
		return nil, err
	}

	// Connect to the kv storage.
	service.StorageMtx = new(sync.Mutex)
	service.Bolt, err = bolt.Open(service.Cfg.Storage, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	// Create storage buckets.
	err = service.createBuckets()
	if err != nil {
		return nil, err
	}

	// Create the server admin.
	_, err = service.CacheGet(util.AdminKey)
	if err != nil {
		if err.Error() == util.ErrKeyNotFound(string(util.AdminKey)).Error() {
			payload := map[string]interface{}{
				"firstName": service.Cfg.Server,
				"lastName":  util.Admin,
				"email":     service.Cfg.AdminEmail,
				"password":  service.Cfg.AdminPass,
			}
			_, err := service.createAdmin(payload)
			if err != nil {
				log.Error(err)
			}
		}
	}

	// Connect to S3.
	service.S3, err = util.NewS3Connection(service.Cfg.AWSAccessKey, service.Cfg.AWSSecretKey, service.Cfg.AWSRegion)
	if err != nil {
		return nil, err
	}

	// Create object storage bucket.
	err = service.S3.CreateBucket(service.Cfg.AWSBucket)
	if err != nil {
		return nil, err
	}

	// Create the http client.
	service.HTTPClient = &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 2,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: time.Second * 10,
	}

	// Create the email client.
	service.MailGun = mailgun.NewMailgun(service.Cfg.MailgunDomain,
		service.Cfg.MailgunAPIKey, service.Cfg.MailgunPublicAPIKey)

	// Create the session map.
	service.SessionMap = cmap.New()

	// Create the router.
	service.Router = new(mux.Router)

	// Load sessions into memory.
	err = service.LoadSessions()
	if err != nil {
		return nil, err
	}

	// Empty the session storage.
	err = service.ClearSessions()
	if err != nil {
		return nil, err
	}

	return service, nil
}

// ValidateRequest asserts the authenticity and requested privileges of a request.
func (service *Service) ValidateRequest(allowedRoles []string, req *http.Request) (bool, error) {
	// Get the session token
	token, err := util.GetSessionToken(req)
	if err != nil {
		return false, err
	}

	// Assert the session token is valid
	ok := service.SessionMap.Has(token)
	if !ok {
		return false, util.ErrUnauthorizedAccess
	}

	// Retrieve the session and assert it has not expired.
	entry, _ := service.SessionMap.Get(token)
	session := entry.(entity.Session)
	if time.Now().Unix() > session.Expiry {
		service.SessionMap.Remove(token)
		return false, util.ErrExpiredSession
	}

	payload := map[string]interface{}{}

	// Parse the request
	req.ParseForm()
	if req.Body != nil {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return false, util.ErrMalformedRequest
		}

		if body != nil {
			err = json.Unmarshal(body, &payload)
			if err != nil {
				return false, util.ErrMalformedPayload
			}
		}

		// Restore the io.ReadCloser to its original state, ie. put back
		// the bytes read
		req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	}

	origin := req.RemoteAddr
	route := req.URL.Path
	requestType := req.Method
	reqLog := entity.RequestLog{
		Origin:      origin,
		Requestor:   token,
		RequestType: requestType,
		Route:       route,
		QueryParams: req.Form.Encode(),
		Payload:     payload,
	}

	logBytes, err := json.Marshal(reqLog)
	if err != nil {
		return false, util.ErrMalformedPayload
	}

	// Log the request.
	service.StorageMtx.Lock()
	defer service.StorageMtx.Unlock()
	err = service.Bolt.Update(func(tx *bolt.Tx) error {
		dateStr := fmtdate.Format(util.DateFormat, time.Now())
		logBucket := tx.Bucket(util.LogBucket)
		dayBucket, err := logBucket.CreateBucketIfNotExists([]byte(dateStr))
		if err != nil {
			return err
		}

		tokenBucket, err := dayBucket.CreateBucketIfNotExists([]byte(token))
		if err != nil {
			return err
		}

		err = tokenBucket.Put(logBytes, nil)
		return err
	})

	// Assert the calling user has the required access for the endpoint.
	granted := false
	switch session.Access {
	case util.Admin:
		granted = true
	default:
		for _, role := range allowedRoles {
			if role == session.Access {
				granted = true
				break
			}
		}
		if !granted {
			err = util.ErrUnauthorizedAccess
		}
	}

	// Extend session expiry by one minute for every successful validation.
	if granted {
		curExpiry := time.Unix(session.Expiry, 0)
		session.Expiry = util.GetFutureTime(curExpiry, 0, 0, 1, 0).Unix()
		service.SessionMap.Set(session.Token, session)
	}

	return granted, err
}

// ClearSessions deletes all kv entries in the session bucket.
func (service *Service) ClearSessions() error {
	// Get all keys.
	tokens := new([][]byte)
	err := service.Bolt.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(util.SessionBucket)
		c := bucket.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			*tokens = append(*tokens, k)
		}
		return nil
	})

	// Iterate through all session kv entries and delete them.
	service.StorageMtx.Lock()
	err = service.Bolt.Update(func(b *bolt.Tx) error {
		bucket := b.Bucket(util.SessionBucket)
		for idx := 0; idx < len(*tokens); idx++ {
			err := bucket.Delete((*tokens)[idx])
			if err != nil {
				return err
			}
		}
		return nil
	})
	service.StorageMtx.Unlock()
	// NB: We are using this approach to get around a bolt bug preventing
	// deletes while iterating with a cursor.
	// See: https://github.com/boltdb/bolt/issues/620
	return err
}

// SaveSessions persists all unexpired sessions.
func (service *Service) SaveSessions() error {
	// Save all unexpired sessions in the in-memory session store.
	service.StorageMtx.Lock()
	err := service.Bolt.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(util.SessionBucket)
		keys := service.SessionMap.Keys()
		now := time.Now().Unix()
		for idx := 0; idx < len(keys); idx++ {
			entry, _ := service.SessionMap.Get(keys[idx])
			session := entry.(entity.Session)

			// Only save unexpired sessions
			if now < session.Expiry {
				sessionBytes, err := json.Marshal(session)
				if err != nil {
					return err
				}
				err = bucket.Put([]byte(keys[idx]), sessionBytes)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	service.StorageMtx.Unlock()
	return err
}

// LoadSessions fetches all unexpired sessions into memory.
func (service *Service) LoadSessions() error {
	// Load all unexpired sessions into the in-memory session store.
	err := service.Bolt.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(util.SessionBucket)
		now := time.Now().Unix()
		c := bucket.Cursor()
		session := new(entity.Session)
		for k, v := c.First(); k != nil; k, v = c.Next() {
			err := json.Unmarshal(v, session)
			if err != nil {
				return err
			}

			// Only load unexpired sessions
			if now > session.Expiry {
				service.SessionMap.Set(string(k), session)
			}
		}
		return nil
	})
	return err
}

func (service *Service) createBuckets() error {
	// Create buckets if they are non-existent.
	service.StorageMtx.Lock()
	err := service.Bolt.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(util.LogBucket)
		if err != nil {
			log.Errorf("failed to create bucket %s", string(util.LogBucket))
			return err
		}

		_, err = tx.CreateBucketIfNotExists(util.InviteBucket)
		if err != nil {
			log.Errorf("failed to create bucket %s", string(util.InviteBucket))
			return err
		}

		_, err = tx.CreateBucketIfNotExists(util.SessionBucket)
		if err != nil {
			log.Errorf("failed to create bucket %s", string(util.SessionBucket))
			return err
		}

		_, err = tx.CreateBucketIfNotExists(util.UserBucket)
		if err != nil {
			log.Errorf("failed to create bucket %s", string(util.UserBucket))
			return err
		}

		_, err = tx.CreateBucketIfNotExists(util.CacheBucket)
		if err != nil {
			log.Errorf("failed to create bucket %s", string(util.CacheBucket))
			return err
		}

		_, err = tx.CreateBucketIfNotExists(util.PassResetBucket)
		if err != nil {
			log.Errorf("failed to create bucket %s", string(util.PassResetBucket))
			return err
		}

		_, err = tx.CreateBucketIfNotExists(util.FeedbackBucket)
		if err != nil {
			log.Errorf("failed to create bucket %s", string(util.FeedbackBucket))
			return err
		}

		return err
	})
	service.StorageMtx.Unlock()
	return err
}

func (service *Service) createAdmin(payload map[string]interface{}) (*entity.User, error) {
	firstName, ok := payload["firstName"].(string)
	if !ok {
		return nil, util.ErrKeyNotFound("firstName")
	}

	lastName, ok := payload["lastName"].(string)
	if !ok {
		return nil, util.ErrKeyNotFound("lastName")
	}

	password, ok := payload["password"].(string)
	if !ok {
		return nil, util.ErrKeyNotFound("password")
	}

	email, ok := payload["email"].(string)
	if !ok {
		return nil, util.ErrKeyNotFound("email")
	}

	hashedPassword, err := util.BcryptHash(password)
	if err != nil {
		return nil, util.ErrBcryptHash
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
		Role:         util.Admin,
		Invite:       "-",
	}

	err = user.Update(service.Bolt, service.StorageMtx)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Cache the admin id.
	err = service.CachePut((util.AdminKey), []byte(user.Uuid))
	if err != nil {
		log.Error(err)
		return nil, err
	}

	user.Sanitize()
	return &user, nil
}

// createRoutes wires up the service routes.
func (service *Service) createRoutes() {
	CreateInviteRoutes(service.Router)
}

// CachePut stores an entity in the server cache.
func (service *Service) CachePut(id []byte, value []byte) error {
	service.StorageMtx.Lock()
	err := service.Bolt.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(util.CacheBucket)
		err := bucket.Put(id, value)
		return err
	})
	service.StorageMtx.Unlock()
	return err
}

// CacheGet retrieves an entity from the server cache.
func (service *Service) CacheGet(id []byte) ([]byte, error) {
	var v []byte
	err := service.Bolt.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(util.CacheBucket)
		v = bucket.Get(id)
		if v == nil {
			return util.ErrKeyNotFound(string(id))
		}

		return nil
	})
	return v, err
}

// Delete removes the specified key and its associated value from storage.
func (service *Service) Delete(bucket, key []byte) error {
	service.StorageMtx.Lock()
	err := service.Bolt.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		return b.Delete(key)
	})
	service.StorageMtx.Unlock()
	return err
}

// Route wires up all API endpoints with their respective handlers.
func (service *Service) SetupRoutes() {
	CreateInviteRoutes(service.Router)
	CreateUserRoutes(service.Router)
	CreateFeedbackRoutes(service.Router)
	CreatePassResetRoutes(service.Router)
	CreateSessionRoutes(service.Router)
}
