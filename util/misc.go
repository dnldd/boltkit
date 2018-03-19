package util

import (
	"fmt"
	"math"
	"net/http"
	"strings"

	mailgun "github.com/mailgun/mailgun-go"
	"golang.org/x/crypto/bcrypt"
)

// Round rounding function.
func Round(f float64, places uint) float64 {
	shift := math.Pow(10, float64(places))
	return math.Floor(f*shift+.5) / shift
}

// BetweenInclusive fetches a substring bounded by two inclusive delimiters.
func BetweenInclusive(value string, a string, b string) string {
	posFirst := strings.Index(value, a)
	if posFirst == -1 {
		return ""
	}
	posLast := strings.Index(value, b)
	if posLast == -1 {
		return ""
	}
	posLastAdjusted := posLast + len(b)
	return value[posFirst:posLastAdjusted]
}

// SendEmail sends an email.
func SendEmail(mailGun mailgun.Mailgun, fromName string, fromEmail string, subject string, body string, to string) error {
	fromFormat := fmt.Sprintf("%s <%s>", fromName, fromEmail)
	message := mailGun.NewMessage(fromFormat, subject, "message", to)
	message.SetHtml(body)
	_, _, err := mailGun.Send(message)
	return err
}

// BcryptHash generates a bcrypt hash from the supplied plaintext.
func BcryptHash(plaintext string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("Failed to hash plaintext: %s", err)
	}
	return string(hashedPassword), nil
}

// GetSessionToken retrieves the session token from a request. The Authorization
// format expected is: 'Token sessiontoken'.
func GetSessionToken(request *http.Request) (string, error) {
	if len(request.Header["Authorization"]) > 0 {
		auth := strings.SplitN(request.Header["Authorization"][0], " ", 2)
		if len(auth) != 2 || auth[0] != "Token" {
			return "", ErrUnexpectedAuthorization
		}
		return auth[1], nil
	}
	return "", ErrAuthorizationNotFound
}

// GetMime returns the MIME type of a file.
func GetMime(data *[]byte) string {
	return http.DetectContentType((*data)[:512])
}
