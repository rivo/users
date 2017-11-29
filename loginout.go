package users

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rivo/sessions"
	"golang.org/x/crypto/bcrypt"
)

// LogIn logs a user into the system, i.e. attaches their User object to the
// current session. Upon a GET request, the "login.gohtml" template is shown
// if no user is logged in yet. If they are logged in (which is checked by
// calling IsLoggedIn()), they are redirected to Config.RouteLoggedIn. A POST
// request will cause a login attempt. After a successful login attempt, users
// are redirected to Config.RouteLoggedIn.
func LogIn(response http.ResponseWriter, request *http.Request) {
	if request.Method == "GET" {
		// If we're already logged in, skip ahead.
		if user, _, _ := IsLoggedIn(response, request); user != nil {
			Config.Log.Printf("Login page visited while logged in with %s (%s)", user.GetID(), user.GetEmail())
			http.Redirect(response, request, Config.RouteLoggedIn, 302)
			return
		}

		// Display a login form.
		RenderPageBasic(response, request, "login.gohtml", nil)
		return
	}

	email := strings.ToLower(request.PostFormValue("email"))
	password := request.PostFormValue("password")

	// Wait a second.
	time.Sleep(time.Second)

	// Load user.
	user, err := Config.LoadUserByEmail(email)
	if err != nil {
		RenderProgramError(response, request, "Could not load user", "", err)
		return
	}
	if user == nil {
		Config.Log.Printf("Non-existing email entered during login: %s", email)
		RenderPageError(response, request, "login.gohtml", "wronglogin", nil, nil)
		return
	}

	// Check password.
	if er := bcrypt.CompareHashAndPassword(user.GetPasswordHash(), []byte(password)); er != nil {
		Config.Log.Printf(`Login password not correct: %s (%s)`, user.GetID(), email)
		RenderPageError(response, request, "login.gohtml", "wronglogin", nil, nil)
		return
	}

	// Is the user in the correct state?
	switch user.GetState() {
	case StateVerified, StateExpired:
	case StateCreated:
		Config.Log.Printf(`Login attempted despite account not yet verified: %s (%s)`, user.GetID(), email)
		RenderPageError(response, request, "signup.gohtml", "verificationincomplete", map[string]string{}, nil)
		return
	default:
		RenderProgramError(response, request, fmt.Sprintf("Unknown user state %d: %s (%s)", user.GetState(), user.GetID(), email), "Invalid user state", nil)
		return
	}

	// Log the user in.
	session, err := sessions.Start(response, request, true)
	if err != nil {
		RenderProgramError(response, request, "Error starting session during login", "Could not start user session", err)
		return
	}
	if err := session.LogIn(user, false, response); err != nil {
		RenderProgramError(response, request, "Login failed", "", err)
		return
	}

	Config.Log.Printf("User %s (%s) was logged in", user.GetID(), email)
	if Config.LoggedIn != nil {
		Config.LoggedIn(user, request.RemoteAddr)
	}
	http.Redirect(response, request, Config.RouteLoggedIn, 302)
}

// IsLoggedIn checks if a user is logged in. If they are, the User object is
// returned. If they aren't logged in, nil is returned. The session object is
// also returned if there was one. (There is always one when a user is
// returned.)
//
// If there was an error or if the user is not in a valid state, an (English)
// error message is returned which is clean enough to be shown to the user.
// Because errors are automatically logged and the returned user for an error
// is nil, it is often ok not to show the error to the user.
//
// Callers will need to check for themselves if the user's state is
// StateExpired, in which case an according message should be displayed. In that
// state, users should not have access to any functionality but instead be
// presented with information instructing them what to do to regain access.
//
// This function will also send HTTP headers that instruct the browser not to
// cache this page.
func IsLoggedIn(response http.ResponseWriter, request *http.Request) (User, *sessions.Session, error) {
	// Never cache this page.
	response.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	response.Header().Set("Pragma", "no-cache")
	response.Header().Set("Expires", "0")

	// Get the session.
	session, err := sessions.Start(response, request, false)
	if err != nil {
		Config.Log.Printf(`Login check failed, could not get session on %s: %s`, request.RequestURI, err)
		return nil, nil, errors.New("Unable to retrieve session")
	}
	if session == nil {
		return nil, nil, nil
	}

	// We have a session. Is a user logged in?
	if session.User() == nil {
		return nil, session, nil
	}

	// We need the correct user state.
	user := session.User().(User)
	switch user.GetState() {
	case StateVerified, StateExpired:
	case StateCreated:
		session.LogOut()
		Config.Log.Printf(`Login check failed because account is not verified: %s (%s) on %s`, user.GetID(), user.GetEmail(), request.RequestURI)
		return nil, session, errors.New("Cannot access this page (verification incomplete)")
	default:
		Config.Log.Printf(`Login check failed because of an unknown user state "%d": %s (%s) on %s`, user.GetState(), user.GetID(), user.GetEmail(), request.RequestURI)
		return nil, session, errors.New("Cannot access this page (unknown user state)")
	}

	return user, session, nil
}

// LogOut logs the user out of the current session. This does not work if it's
// a GET request (i.e. a simple link) because URL pre-fetching or proxies may
// cause users to log out. A simple form with a button will cause the logout
// link to be visited using POST:
//
//   <form action="/logout" method="POST"><button>Log out</button></form>
//
// You can use CSS to make the button look like a link.
func LogOut(response http.ResponseWriter, request *http.Request) {
	// Make sure we only process POST requests.
	if request.Method != "POST" {
		RenderProgramError(response, request, "Logout request method was "+request.Method, "Logout method must be POST", nil)
		return
	}

	// Get the session.
	session, err := sessions.Start(response, request, false)
	if err != nil {
		RenderProgramError(response, request, "Error starting session during logout", "Could not start user session", err)
		return
	}

	// If there is no session or if no user is attached to it,
	// the user is already logged out.
	if session == nil || session.User() == nil {
		Config.Log.Print("Logout requested when user is already logged out")
		http.Redirect(response, request, Config.RouteLoggedOut, 302)
		return
	}

	// Log the user out.
	user := session.User().(User)
	id := user.GetID()
	email := user.GetEmail()
	if err := session.LogOut(); err != nil {
		RenderProgramError(response, request, "Could not log user out of session", "", err)
		return
	}
	if err := session.RegenerateID(response); err != nil {
		RenderProgramError(response, request, "Could not regenerate session ID", "", err)
		return
	}

	// Destroying the session is optional.
	if err := session.Destroy(response, request); err != nil {
		RenderProgramError(response, request, "Error destroying session during logout", "Could not destroy user session", err)
		return
	}

	Config.Log.Printf("User %s (%s) was logged out", id, email)
	http.Redirect(response, request, Config.RouteLoggedOut, 302)
}
