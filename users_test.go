package users

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rivo/sessions"
)

// ExampleUser implements the User interface.
type ExampleUser struct {
	id             string
	email          string
	passwordHash   []byte
	state          int
	verificationID string
	vidCreated     time.Time
	passwordToken  string
	tokenCreated   time.Time
}

func (u *ExampleUser) GetID() interface{} {
	return u.id
}

func (u *ExampleUser) SetID(id interface{}) {
	u.id = id.(string)
}

func (u *ExampleUser) SetState(state int) {
	u.state = state
}

func (u *ExampleUser) GetState() int {
	return u.state
}

func (u *ExampleUser) SetEmail(email string) {
	u.email = email
}

func (u *ExampleUser) GetEmail() string {
	return u.email
}

func (u *ExampleUser) SetPasswordHash(hash []byte) {
	u.passwordHash = hash
}

func (u *ExampleUser) GetPasswordHash() []byte {
	return u.passwordHash
}

func (u *ExampleUser) SetVerificationID(id string, created time.Time) {
	u.verificationID = id
	u.vidCreated = created
}

func (u *ExampleUser) GetVerificationID() (string, time.Time) {
	return u.verificationID, u.vidCreated
}

func (u *ExampleUser) SetPasswordToken(id string, created time.Time) {
	u.passwordToken = id
	u.tokenCreated = created
}

func (u *ExampleUser) GetPasswordToken() (string, time.Time) {
	return u.passwordToken, u.tokenCreated
}

func (u *ExampleUser) GetRoles() []string {
	return nil
}

func Example() {
	//  We need a way to create new users.
	Config.NewUser = func() User {
		return &ExampleUser{
			id: sessions.CUID(),
		}
	}

	// Set a starting point for when users have just logged in.
	Config.RouteLoggedIn = "/start"

	// Add a handler for the start page.
	http.HandleFunc(Config.RouteLoggedIn, func(response http.ResponseWriter, request *http.Request) {
		// Is a user logged in?.
		if user, _, _ := IsLoggedIn(response, request); user != nil {
			if user == nil {
				fmt.Fprint(response, "<body>No user is logged in</body>")
				return
			}

			// Yes, a user is logged in.
			fmt.Fprintf(response, "<body>User %s (%s) is logged in", user.GetID(), user.GetEmail())
			if user.GetState() == StateExpired {
				fmt.Fprint(response, ", but expired")
			}
			fmt.Fprintf(response, ` <form action="%s" method="POST"><button>Log out</button></form></body>`, Config.RouteLogOut)
			return
		}

		fmt.Fprint(response, "<body>No user is logged in</body>")
	})

	// Start the server.
	if err := Main(); err != nil {
		Config.Log.Printf("Server execution failed: %s", err)
	}
}
