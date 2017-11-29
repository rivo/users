package users

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rivo/sessions"
	"golang.org/x/crypto/bcrypt"
)

// ForgottenPassword renders the "forgottenpassword.gohtml" template upon a GET
// request, unless a user is logged in (checked with IsLoggedIn()), in which
// case they are redirected to Config.RouteLoggedIn. Upon a POST request, an
// email is sent to the provided address. If the email address is of an existing
// user account, a temporary ID for a password reset link is generated and the
// link sent in the email (using the "reset_existing.tmpl" mail template). If
// the email address is unknown, the email sent will contain basic information
// about the request (using the "reset_unknown.tmpl" mail template). In any
// case, the "resetlinksent.gohtml" template is rendered.
func ForgottenPassword(response http.ResponseWriter, request *http.Request) {
	if request.Method == "GET" {
		user, _, _ := IsLoggedIn(response, request)
		if user != nil {
			// A user is already logged in. Abort.
			Config.Log.Printf("Forgotten password link visited while logged in with %s (%s)", user.GetID(), user.GetEmail())
			http.Redirect(response, request, Config.RouteLoggedIn, 302)
			return
		}

		// Just render the "forgotten password" page.
		RenderPageBasic(response, request, "forgottenpassword.gohtml", nil)
		return
	}

	// Check if we know this user.
	email := strings.ToLower(request.PostFormValue("email"))
	user, err := Config.LoadUserByEmail(email)
	if err != nil {
		RenderProgramError(response, request, "Could not load user on forgotten password: "+email, "Could not load user", err)
		return
	}

	// Check what needs to be done now.
	template := "reset_existing.tmpl"
	data := map[string]interface{}{
		"email":  email,
		"date":   time.Now().Format("Mon, 2006-01-02 15:04:05"),
		"ip":     request.RemoteAddr,
		"agent":  request.UserAgent(),
		"config": Config,
		"user":   user,
	}
	if user != nil && user.GetState() == StateVerified {
		// The user exists and is verified. Create a reset ID.
		token, err := sessions.RandomID(22)
		if err != nil {
			RenderProgramError(response, request, fmt.Sprintf("Could not generate password reset ID for %s (%s)", user.GetID(), user.GetEmail()), "Could not generate password reset ID", err)
			return
		}
		tokenCreated := time.Now()
		user.SetPasswordToken(token, tokenCreated)
		if err := Config.UpdateUser(user); err != nil {
			RenderProgramError(response, request, fmt.Sprintf("Cannot save user with new password reset ID: %s (%s)", user.GetID(), user.GetEmail()), "Cannot update user", err)
			return
		}
		data["token"] = token
		data["validity"] = tokenCreated.Add(24 * time.Hour).Format("Monday, Jan 2, 2006, 15:04:05")
		Config.Log.Printf("Sending password reset email for existing account: %s (%s)", user.GetID(), user.GetEmail())
	} else {
		// This user does not exist
		template = "reset_unknown.tmpl"
		Config.Log.Printf("Sending passwort reset info email for unknown account: %s", email)
	}

	// Send password reset email.
	if err := SendMail(request, email, template, data); err != nil {
		RenderProgramError(response, request, "Could not send password reset email", "", err)
		return
	}

	RenderPage(response, request, "resetlinksent.gohtml", map[string]interface{}{"email": email})
}

// ResetPassword checks, upon a GET request, the provided token and renders
// the "resetpassword.gohtml" template which contains a form to reset the user's
// password. Upon a POST request, the entered password is checked and saved.
// Upon success, the user is logged out of all sessions (provided
// sessions.Persistence.UserSessions is implemented) and the
// "passwordreset.gohtml" template is shown.
func ResetPassword(response http.ResponseWriter, request *http.Request) {
	// Check if we have a valid password reset token.
	token := request.FormValue("token")
	user, err := Config.LoadUserByPasswordToken(token)
	if err != nil {
		RenderProgramError(response, request, "Could not load user via password reset token: "+token, "Could not load user", err)
		return
	}
	if user == nil {
		Config.Log.Printf("Reset password token unknown: %s", token)
		RenderPageError(response, request, "forgottenpassword.gohtml", "resettokennotfound", nil, nil)
		return
	}
	_, tokenCreated := user.GetPasswordToken()
	if tokenCreated.Add(30 * time.Minute).Before(time.Now()) {
		Config.Log.Printf("Password reset token for user %s (%s) expired: %s", user.GetID(), user.GetEmail(), token)
		RenderPageError(response, request, "forgottenpassword.gohtml", "resettokenexpired", nil, nil)
		return
	}

	if request.Method == "GET" {
		// Simply render the reset password template.
		RenderPage(response, request, "resetpassword.gohtml", map[string]interface{}{"config": Config, "infos": map[string]string{"token": token}})
		return
	}

	// Get the form entries.
	password := request.PostFormValue("password")
	passwordConfirm := request.PostFormValue("passwordconfirm")

	// Check if passwords match.
	if password != passwordConfirm {
		Config.Log.Printf("New passwords for %s (%s) don't match", user.GetID(), user.GetEmail())
		RenderPageError(response, request, "resetpassword.gohtml", "passwordsdontmatch", map[string]string{"token": token}, nil)
		return
	}

	// Check password integrity.
	if result := sessions.ReasonablePassword(password, append(Config.PasswordNames, user.GetEmail())); result != sessions.PasswordOK {
		Config.Log.Printf("New password was rejected for %s (%s), reason: %d", user.GetID(), user.GetEmail(), result)
		RenderPageError(response, request, "resetpassword.gohtml", "invalidpassword", map[string]interface{}{"issue": result, "token": token}, nil)
		return
	}

	// Generate password hash.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 0)
	if err != nil {
		RenderProgramError(response, request, "Could not generate new password hash", "", err)
		return
	}

	// Save new password.
	user.SetPasswordHash(hash)
	user.SetPasswordToken("", time.Unix(0, 0)) // Invalidate token.
	if err := Config.UpdateUser(user); err != nil {
		RenderProgramError(response, request, "Could not save user with new password", "", err)
		return
	}
	Config.Log.Printf("Password was reset for user %s (%s)", user.GetID(), user.GetEmail())

	// Log the user out of all sessions.
	if err := sessions.LogOut(user.GetID()); err != nil {
		RenderProgramError(response, request, "Could not log user out of all sessions", "", err)
		return
	}

	// Show a confirmation.
	RenderPageBasic(response, request, "passwordreset.gohtml", nil)
}
