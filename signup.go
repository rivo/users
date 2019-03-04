package users

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/rivo/sessions"
)

// SignUp renders the "signup.gohtml" template upon a GET request, unless the
// user is already logged in (which is checked using IsLoggedIn()), in which
// case they are redirected to Config.RouteLoggedIn. On a POST request, it
// attempts to create a new user given an email address, a password, and its
// confirmation. A successful sign-up will lead to a validation email to be sent
// (using the "verification_new.tmpl" mail template for new users and the
// "verification_existing.tmpl" mail template for existing users) and the
// "validationsent.gohtml" template to be shown.
func SignUp(response http.ResponseWriter, request *http.Request) {
	if request.Method == "GET" {
		user, _, _ := IsLoggedIn(response, request)
		if user != nil {
			Config.Log.Printf("Sign-up page visited while logged in with %s (%s)", user.GetID(), user.GetEmail())
			http.Redirect(response, request, Config.RouteLoggedIn, 302)
			return
		}

		RenderPageBasic(response, request, "signup.gohtml", nil)
		return
	}

	// Get the form entries.
	email := strings.ToLower(request.PostFormValue("email"))
	password := request.PostFormValue("password")
	passwordConfirm := request.PostFormValue("passwordconfirm")

	// Perform a very basic email check. We'll send a validation email anyway.
	if !strings.Contains(email, "@") {
		RenderPageError(response, request, "signup.gohtml", "invalidemail", map[string]string{"email": email}, nil)
		return
	}

	// Check if passwords match.
	if password != passwordConfirm {
		Config.Log.Printf("Passwords for %s don't match", email)
		RenderPageError(response, request, "signup.gohtml", "passwordsdontmatch", map[string]string{"email": email}, nil)
		return
	}

	// Check password integrity.
	if result := sessions.ReasonablePassword(password, append(Config.PasswordNames, email)); result != sessions.PasswordOK {
		Config.Log.Printf("Password was rejected for %s, reason: %d", email, result)
		RenderPageError(response, request, "signup.gohtml", "invalidpassword", map[string]interface{}{"email": email, "issue": result}, nil)
		return
	}

	// Generate password hash.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 0)
	if err != nil {
		RenderProgramError(response, request, "Could not generate password hash", "", err)
		return
	}

	// Make sure we have a NewUser function.
	if Config.NewUser == nil {
		RenderProgramError(response, request, "NewUser is not implemented", "", err)
		return
	}

	// Create a new user.
	user := Config.NewUser()
	verificationID, err := sessions.RandomID(22)
	if err != nil {
		RenderProgramError(response, request, "Unable to create a verification ID", "", err)
		return
	}
	idCreated := time.Now()
	user.SetVerificationID(verificationID, idCreated)
	user.SetState(StateCreated)
	user.SetEmail(email)
	user.SetPasswordHash(hash)

	// Save that new user.
	existingUser, err := Config.SaveNewUserAtomic(user)
	if err != nil {
		RenderProgramError(response, request, "Error saving new user", "", err)
		return
	}

	// Check what needs to be done now.
	template := "verification_new.tmpl"
	if existingUser != nil {
		// We already have this user in our database. What we do now depends on
		// their state.
		switch existingUser.GetState() {
		case StateVerified, StateExpired:
			// Don't verify again. We send a notification of this creation attempt.
			template = "verification_existing.tmpl"
			Config.Log.Printf("Sending verification notification for existing account: %s (%s)", existingUser.GetID(), email)
		case StateCreated:
			// This user was already created but not yet verified. Refresh the
			// verification ID.
			user.SetID(existingUser.GetID())
			if err := Config.UpdateUser(user); err != nil {
				RenderProgramError(response, request, fmt.Sprintf("Cannot refresh verification ID for user %s (%s)", user.GetID(), email), "Error exchanging verification ID", err)
				return
			}
			Config.Log.Printf("Sending repeated verification email for new account: %s (%s)", user.GetID(), email)
		default:
			RenderProgramError(response, request, fmt.Sprintf("Unknown user state %d: %s (%s)", user.GetState(), user.GetID(), email), "Invalid user state", nil)
			return
		}
	} else {
		// This user is new and needs to be verified.
		Config.Log.Printf("Sending verification email for new account: %s (%s)", user.GetID(), email)
	}

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
		RenderProgramError(response, request, "Could not send verification email", "", err)
		return
	}

	RenderPage(response, request, "verificationsent.gohtml", map[string]interface{}{"config": Config, "email": email})
}

// Verify processes a verification link by checking the provided verification ID
// and, if valid, setting the user's state to "verified".
func Verify(response http.ResponseWriter, request *http.Request) {
	if Config.ThrottleVerification != nil {
		Config.ThrottleVerification()
	}

	// Find the user for this verification ID.
	verificationID := request.FormValue("id")
	user, err := Config.LoadUserByVerificationID(verificationID)
	if err != nil {
		RenderProgramError(response, request, "Could not load user for verification ID", "", err)
		return
	}
	if user == nil {
		Config.Log.Printf("Verification ID not found: %s", verificationID)
		RenderPageError(response, request, "signup.gohtml", "verificationidnotfound", map[string]string{}, nil)
		return
	}

	// Do we have the right user?
	userVerificationID, idCreated := user.GetVerificationID()
	if userVerificationID != verificationID {
		RenderProgramError(
			response,
			request,
			fmt.Sprintf("Wrong user loaded: %s (%s) with verification ID %s instead of %s", user.GetID(), user.GetEmail(), userVerificationID, verificationID),
			"Wrong user loaded",
			err,
		)
		return
	}

	// Is the verification ID still valid?
	if idCreated.Add(3 * 24 * time.Hour).Before(time.Now()) {
		Config.Log.Printf("Verification ID for user %s (%s) expired: %s", user.GetID(), user.GetEmail(), verificationID)
		RenderPageError(response, request, "signup.gohtml", "verificationidexpired", map[string]string{}, nil)
		return
	}

	// User has been verified. Update status.
	user.SetState(StateVerified)
	user.SetVerificationID("", time.Unix(0, 0)) // Invalidate verification ID.
	if err = Config.UpdateUser(user); err != nil {
		RenderProgramError(response, request, fmt.Sprintf("Could not verify user %s (%s)", user.GetID(), user.GetEmail()), "Could not verify user", err)
		return
	}
	Config.Log.Printf("User %s (%s) has been verified", user.GetID(), user.GetEmail())

	// If anyone is logged in, log them out now.
	session, _ := sessions.Start(response, request, false)
	if session != nil && session.User() != nil {
		session.LogOut()
	}

	// Show a confirmation.
	RenderPageBasic(response, request, "verified.gohtml", nil)
}
