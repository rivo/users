package users

import (
	"net/http"
	"strings"
	"time"

	"github.com/rivo/sessions"
	"golang.org/x/crypto/bcrypt"
)

// Change returns, upon a GET request, the "changeinfos.gohtml" template which
// allows a user to change their email address and/or password. Upon a POST
// request, any values that have been modified by the user are changed and will
// undergo the same procedure as a signup (e.g. verification for email changes).
// The "infoschanged.gohtml" template will be used for confirmation.
//
// Any of this only works if a user is currently logged in (checked with
// IsLoggedIn()).
//
// If there are more user attributes that need to be changed than just email and
// password, it makes sense to make a copy of this function and extend it to
// your needs.
func Change(response http.ResponseWriter, request *http.Request) {
	// This page only works if the user is logged in.
	user, session, _ := IsLoggedIn(response, request)
	if user == nil {
		RenderProgramError(response, request, "This page may only be accessed when you are logged in", "", nil)
		return
	}

	// Do we simply render the page?
	if request.Method == "GET" {
		RenderPage(response, request, "changeinfos.gohtml", map[string]interface{}{"config": Config, "user": user, "infos": map[string]string{"email": user.GetEmail()}})
		return
	}

	// Get form input.
	email := strings.ToLower(request.PostFormValue("email"))
	currentPassword := request.PostFormValue("currentpassword")
	password := request.PostFormValue("password")
	passwordConfirm := request.PostFormValue("passwordconfirm")
	emailChanged := email != user.GetEmail()
	passwordChanged := password != ""

	// Some more variables that will hold the new values.
	var (
		hash           []byte
		verificationID string
		idCreated      time.Time
		emailExists    bool
	)

	// If nothing has changed, we're done.
	if !emailChanged && !passwordChanged {
		RenderPage(response, request, "changeinfos.gohtml", map[string]interface{}{"config": Config, "user": user, "infos": map[string]string{"email": user.GetEmail()}})
		return
	}

	// Validate the current password. It needs to be provided for any changes.
	if currentPassword == "" {
		Config.Log.Printf("User %s (%s) tried to make changes, current password not provided", user.GetID(), user.GetEmail())
		RenderPageError(response, request, "changeinfos.gohtml", "currentpasswordnotprovided", map[string]string{"email": email}, user)
		return
	}
	if err := bcrypt.CompareHashAndPassword(user.GetPasswordHash(), []byte(currentPassword)); err != nil {
		Config.Log.Printf("User %s (%s) tried to make changes, current password wrong", user.GetID(), user.GetEmail())
		Config.Log.Printf("User: %v", user)
		RenderPageError(response, request, "changeinfos.gohtml", "currentpasswordwrong", map[string]string{"email": email}, user)
		return
	}

	// Does the user want to change their password?
	if passwordChanged {
		// Check if passwords match.
		if password != passwordConfirm {
			Config.Log.Printf("Changed passwords for %s (%s) don't match", user.GetID(), user.GetEmail())
			RenderPageError(response, request, "changeinfos.gohtml", "passwordsdontmatch", map[string]string{"email": email}, user)
			return
		}

		// Check password integrity.
		if result := sessions.ReasonablePassword(password, append(Config.PasswordNames, user.GetEmail())); result != sessions.PasswordOK {
			Config.Log.Printf("Changed password was rejected for %s (%s), reason: %d", user.GetID(), user.GetEmail(), result)
			RenderPageError(response, request, "changeinfos.gohtml", "invalidpassword", map[string]interface{}{"issue": result, "email": email}, user)
			return
		}

		// Generate password hash.
		var err error
		hash, err = bcrypt.GenerateFromPassword([]byte(password), 0)
		if err != nil {
			RenderProgramError(response, request, "Could not generate changed password hash", "", err)
			return
		}

		Config.Log.Printf("Password was changed for user %s (%s)", user.GetID(), user.GetEmail())
	}

	// Does the user want to change their email address?
	if emailChanged {
		// Perform a very basic email check. We'll send a validation email anyway.
		if !strings.Contains(email, "@") {
			Config.Log.Printf("User %s (%s) tried to make changes, new email invalid: %s", user.GetID(), user.GetEmail(), email)
			RenderPageError(response, request, "changeinfos.gohtml", "invalidemail", map[string]string{"email": email}, user)
			return
		}

		// Check if there is a user with the new email address?
		existingUser, err := Config.LoadUserByEmail(email)
		if err != nil {
			RenderProgramError(response, request, "Could not check email validity", "", err)
			return
		}

		// Again, to avoid providing a method that allows someone to find out if
		// a user account with a specific email address exists, the response is
		// always the same.

		// Check what needs to be done now.
		template := "verification_changed.tmpl"
		if existingUser != nil {
			// We already have this user in our database. We won't change the email
			// address but the user will be set back to unverified.
			emailExists = true
			template = "verification_existing.tmpl"
			Config.Log.Printf("Sending verification notification for existing account upon email change: %s (%s)", existingUser.GetID(), email)
		} else {
			// This email address wasn't known yet. Verify the user.
			Config.Log.Printf("Sending verification email for new account: %s (%s)", user.GetID(), email)
		}

		// Set a verification ID.
		verificationID, err = sessions.RandomID(22)
		if err != nil {
			RenderProgramError(response, request, "Could not generate verification ID", "", err)
			return
		}
		idCreated = time.Now()

		// Send notification email.
		data := map[string]interface{}{
			"email":        email,
			"date":         time.Now().Format("Mon, 2006-01-02 15:04:05"),
			"ip":           request.RemoteAddr,
			"agent":        request.UserAgent(),
			"verification": verificationID,
			"validity":     idCreated.Add(3 * 24 * time.Hour).Format("Monday, Jan 2, 2006, 15:04:05"),
			"config":       Config,
			"user":         user,
		}
		if err := SendMail(request, email, template, data); err != nil {
			RenderProgramError(response, request, "Could not send email change verification email", "", err)
			return
		}

		Config.Log.Printf("Email address was changed for user %s (%s) to %s", user.GetID(), user.GetEmail(), email)
	}

	// All checks were successful. Modify and save the new user.
	if passwordChanged {
		user.SetPasswordHash(hash)
	}
	if emailChanged {
		if !emailExists {
			user.SetEmail(email)
		}
		user.SetState(StateCreated)
		user.SetVerificationID(verificationID, idCreated)
	}
	if err := Config.UpdateUser(user); err != nil {
		RenderProgramError(response, request, "Could not save user with changes", "", err)
		return
	}

	// Update this user in all sessions.
	if err := sessions.RefreshUser(user); err != nil {
		RenderProgramError(response, request, "Could not refresh user with changes", "", err)
		return
	}

	// If email changed, log them out.
	if emailChanged {
		if err := sessions.LogOut(user.GetID()); err != nil {
			RenderProgramError(response, request, "Could not log user out of all sessions", "", err)
			return
		}
		if err := session.LogOut(); err != nil {
			RenderProgramError(response, request, "Could not log user out of current session", "", err)
			return
		}
		user = nil
	}

	RenderPageBasic(response, request, "infoschanged.gohtml", user)
}
