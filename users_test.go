package users

/*
import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rivo/sessions"
	"github.com/rivo/sessions/users"
)

// MyUser implements the users.User interface.
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

func Example() {
	//  We need a way to create new users.
	users.Config.NewUser = func() users.User {
		return &MyUser{
			id: sessions.CUID(),
		}
	}

	// Set a starting point for when users have just logged in.
	users.Config.RouteLoggedIn = "/start"

	// Add a handler for the start page.
	http.HandleFunc(users.Config.RouteLoggedIn, func(response http.ResponseWriter, request *http.Request) {
		// Is a user logged in?.
		if user, _, out := users.IsLoggedIn(response, request); user != nil {
			if out {
				return // Some error message was already sent to the browser.
			}

			// Yes, a user is logged in.
			fmt.Fprintf(response, "<body>User %s (%s) is logged in", user.GetID(), user.GetEmail())
			if user.GetState() == users.StateExpired {
				fmt.Fprint(response, ", but expired")
			}
			fmt.Fprintf(response, ` <form action="%s" method="POST"><button>Log out</button></form></body>`, users.Config.RouteLogOut)
			return
		}

		fmt.Fprint(response, "<body>No user is logged in</body>")
	})

	// Start the server.
	if err := users.Main(); err != nil {
		Config.Log.Printf("Server execution failed: %s", err)
	}
}

*/
