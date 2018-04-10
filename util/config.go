package util

import "encoding/json"

// Config represents the server configuration file.
type Config struct {
	Port                string `json:"port"`
	Debug               bool   `json:"debug"`
	Server              string `json:"server"`
	HTTPS               bool   `json:"https"`
	Storage             string `json:"storage"`
	AdminEmail          string `json:"adminemail"`
	AdminPass           string `json:"adminpass"`
	ResetEmail          string `json:"resetemail"`
	InviteEmail         string `json:"inviteemail"`
	FeedbackEmail       string `json:"feedbackemail"`
	MailgunAPIKey       string `json:"mailgunapikey"`
	MailgunDomain       string `json:"mailgundomain"`
	MailgunPublicAPIKey string `json:"mailgunpublicapikey"`
	PageLimit           uint32 `json:"pagelimit"`
	Frontend            string `json:"frontend"`
	AWSAccessKey        string `json:"awsaccesskey"`
	AWSSecretKey        string `json:"awssecretkey"`
	AWSRegion           string `json:"awsregion"`
	AWSBucket           string `json:"awsbucket"`
	// MinioEndpoint       string `json:"minioendpoint"`
	// MinioAccessKey      string `json:"minioaccesskey"`
	// MinioSecretKey      string `json:"miniosecretkey"`
	// MinioRegion         string `json:"minioregion"`
	// MinioBucketName string `json:"miniobucketname"`
}

// NewConfig loads the server configuration file.
func NewConfig(filepath string) (*Config, error) {
	cfg := new(Config)
	data, err := ReadFileAsBytes(filepath)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		log.Errorf("failed to load server config: %s", err)
	}
	return cfg, err
}
