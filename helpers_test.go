package users

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rivo/sessions"
)

type MyUser struct {
	id             string
	email          string
	passwordHash   []byte
	state          int
	verificationID string
	vidCreated     time.Time
	passwordToken  string
	tokenCreated   time.Time
}

func (u *MyUser) GetID() interface{} {
	return u.id
}

func (u *MyUser) SetID(id interface{}) {
	u.id = id.(string)
}

func (u *MyUser) SetState(state int) {
	u.state = state
}

func (u *MyUser) GetState() int {
	return u.state
}

func (u *MyUser) SetEmail(email string) {
	u.email = email
}

func (u *MyUser) GetEmail() string {
	return u.email
}

func (u *MyUser) SetPasswordHash(hash []byte) {
	u.passwordHash = hash
}

func (u *MyUser) GetPasswordHash() []byte {
	return u.passwordHash
}

func (u *MyUser) SetVerificationID(id string, created time.Time) {
	u.verificationID = id
	u.vidCreated = created
}

func (u *MyUser) GetVerificationID() (string, time.Time) {
	return u.verificationID, u.vidCreated
}

func (u *MyUser) SetPasswordToken(id string, created time.Time) {
	u.passwordToken = id
	u.tokenCreated = created
}

func (u *MyUser) GetPasswordToken() (string, time.Time) {
	return u.passwordToken, u.tokenCreated
}

func (u *MyUser) GetRoles() []string {
	return nil
}

func TestMain(m *testing.M) {
	Config.HTMLTemplateDir = "test"
	Config.MailTemplateDir = "test"
	Config.Log = log.New(ioutil.Discard, "", 0)
	Config.NewUser = func() User {
		return &MyUser{
			id: sessions.CUID(),
		}
	}
	sessions.AcceptChangingUserAgent = true
	os.Exit(m.Run())
}

// Runs a request on the given handler and returns the results. The "get" and
// "post" arguments may be nil. If "post" is not nil, the request method will be
// POST, otherwise GET. If "user" is not nil, they are logged in first.
//
// The function returns the response body as well as the email text that was
// sent (if any).
func runRequest(user User, get, post map[string]string, handler func(response http.ResponseWriter, request *http.Request)) (string, string) {
	// Make HTTP request.
	method := "GET"
	var reqBody string
	u := "/test"
	if get != nil {
		values := url.Values{}
		for key, value := range get {
			values.Add(key, value)
		}
		u += "?" + values.Encode()
	}
	if post != nil {
		method = "POST"
		values := url.Values{}
		for key, value := range post {
			values.Add(key, value)
		}
		reqBody = values.Encode()
	}
	request := httptest.NewRequest(method, u, strings.NewReader(reqBody))
	if post != nil {
		request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	// Make HTTP response.
	response := httptest.NewRecorder()

	// Log in user if required.
	if user != nil {
		sessions.PurgeSessions()
		request.AddCookie(&http.Cookie{
			Name:  "id",
			Value: "01234567890123456789----",
		})
		sessions.Persistence = sessions.ExtendablePersistenceLayer{
			LoadSessionFunc: func(id string) (*sessions.Session, error) {
				now := time.Now().Format(time.RFC3339)
				serialized := fmt.Sprintf(`{"cr":"%s","da":{},"ip":"192.168.178.1:80","la":"%s","rf":"","ua":"0","us":"abcd","v":1}`, now, now)
				session := &sessions.Session{}
				if err := json.Unmarshal([]byte(serialized), session); err != nil {
					panic(err)
				}
				return session, nil
			},
			LoadUserFunc: func(id interface{}) (sessions.User, error) {
				return user, nil
			},
		}
	}

	// Set up email handler.
	var email string
	Config.SendEmails = true
	Config.SendEmail = func(recipient, subject, body string) error {
		email = body
		return nil
	}

	// Run the handler.
	handler(response, request)

	Config.SendEmails = false

	if response.Code == 302 {
		return "redirect", email
	}
	return response.Body.String(), strings.TrimSpace(email)
}

// Outputs an error if the expected and computed strings don't match.
func assertString(expected, computed string, t *testing.T) {
	t.Helper()
	if expected != computed {
		t.Errorf(`Expected "%s" but is "%s"`, expected, computed)
	}
}
