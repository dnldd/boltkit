package util

// Bucket names.
var (
	SessionBucket   = []byte("session")
	UserBucket      = []byte("user")
	InviteBucket    = []byte("invite")
	CacheBucket     = []byte("cache")
	PassResetBucket = []byte("passreset")
	FeedbackBucket  = []byte("feedback")
)

// Cache keys.
var (
	AdminKey = []byte("admin")
)

// Access Privilige types.
var (
	Admin      = "admin"
	Management = "management"
	Finance    = "finance"
)

// Scheduled Job types.
var (
	InviteJob    = "invite"
	PassResetJob = "passreset"
)
