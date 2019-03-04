package users

import (
	"log"
	"os"
	"sync"
	"time"
)

// Config contains all the settings and helper functions needed to run this
// package's code without any modifications. You will need to change many of
// these default values to run the code in this package.
var Config = struct {
	// The address the HTTP server binds to.
	ServerAddr string

	// The logger to which messages produced in this package are sent.
	Log *log.Logger

	// A list of names to exclude from passwords. This is typically the name of
	// your application, its domain name etc.
	PasswordNames []string

	// Routes.
	RouteSignUp            string // The signup page.
	RouteVerify            string // The page where the user verifies their email address.
	RouteLogIn             string // The page with the log-in form.
	RouteLoggedIn          string // The page to which the user is redirected after logging in.
	RouteLogOut            string // The logout page.
	RouteLoggedOut         string // The page to which the user is redirected after logging out.
	RouteForgottenPassword string // The forgotten password page.
	RouteResetPassword     string // The page where the user can choose a new password.
	RouteChange            string // The page where the user can change their email address and/or password.

	// Template settings.
	CacheTemplates       bool // If true, templates are cached after their first use.
	HTMLTemplateDir      string
	HTMLTemplateIncludes []string // Any HTML templates which may be included by other templates.
	MailTemplateDir      string
	MailTemplateIncludes []string // Any mail templates which may be included by other templates.

	// If this value is set to true, the functions in this package will read the
	// value of the user's "lang" cookie and, provided it is a valid language code
	// such as "en" or "en-US", will access the templates in the subdirectory
	// of HTMLTemplateDir and MailTemplateDir with the name of the language code.
	// If the value is false, such a code could not be determined, or the
	// directory does not exist, no subdirectories are used.
	Internationalization bool

	// Email related settings.
	SendEmails   bool
	SendEmail    func(recipient, subject, body string) error // If provided, the following email parameters are ignored.
	SenderName   string
	SenderEmail  string
	SMTPHostname string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string

	// User functions. The default implementations provided here use a local,
	// RAM-based store which is lost when the program stops. They are not very
	// efficient and will slow down on a large number of users. Replace these with
	// your own implementation. Note that importing this package also causes
	// sessions.Persistence.(ExtendablePersistenceLayer).LoadUserFunc to be set
	// to our default implementation.

	// NewUser returns a new user object. For the purposes of this package, only
	// a user ID needs to be set. Other fields will be populated by this package.
	//
	// The default is a nil function so it must be implemented by users of this
	// package.
	NewUser func() User

	// SaveNewUserAtomic saves a new user to the database. If a user with the same
	// email address previously existed, that existing user is returned (and the
	// new user is not saved). If no such user previously existed, they are saved
	// and a nil interface is returned.
	//
	// Checking for the existence of a user and inserting them needs to be an
	// atomic transaction to avoid the duplication of users due to race
	// conditions.
	SaveNewUserAtomic func(user User) (User, error)

	// UpdateUser updates an existing user (identified by their user ID) in the
	// database.
	UpdateUser func(user User) error

	// LoadUserByVerificationID loads a user given a verification ID. If no user
	// was found, it's not an error, just nil is returned.
	LoadUserByVerificationID func(id string) (User, error)

	// LoadUserByPasswordToken loads a user given a password token. If no user
	// was found, it's not an error, just nil is returned.
	LoadUserByPasswordToken func(token string) (User, error)

	// LoadUserByEmail loads a user given their email address. If no user was
	// found, it's not an error, just nil is returned.
	LoadUserByEmail func(email string) (User, error)

	// LoggedIn is called when a user was successfully logged in from a browser
	// at the given IP address.
	LoggedIn func(user User, ipAddress string)

	// ThrottleVerification throttles verification attempts. The default
	// implementation simply pauses all verification requests by one second.
	ThrottleVerification func()

	// ThrottleLogin throttles login attempts. The default implementation pauses
	// each login request by the same user for one second.
	ThrottleLogin func(email string)
}{
	ServerAddr:             ":5050",
	Log:                    log.New(os.Stdout, "", log.LstdFlags),
	PasswordNames:          []string{"example.com", "ExampleCom", "Example"},
	RouteSignUp:            "/signup",
	RouteVerify:            "/verify",
	RouteLogIn:             "/login",
	RouteLoggedIn:          "/",
	RouteLogOut:            "/logout",
	RouteLoggedOut:         "/login",
	RouteForgottenPassword: "/forgottenpassword",
	RouteResetPassword:     "/resetpassword",
	RouteChange:            "/changeinfos",
	CacheTemplates:         false,
	HTMLTemplateDir:        "src/github.com/rivo/sessions/users/html",
	HTMLTemplateIncludes:   []string{"header.gohtml", "footer.gohtml"},
	MailTemplateDir:        "src/github.com/rivo/sessions/users/mail",
	MailTemplateIncludes:   []string{"header.tmpl", "footer.tmpl"},
	Internationalization:   false,
	SendEmails:             false,
	SendEmail:              nil,
	SenderName:             "Example.com Support",
	SenderEmail:            "support@example.com",
	SMTPHostname:           "mail.example.com",
	SMTPPort:               25,
	SMTPUsername:           "support@example.com",
	SMTPPassword:           "password",
	NewUser:                nil,
	SaveNewUserAtomic: func(user User) (User, error) {
		usersMutex.Lock()
		defer usersMutex.Unlock()
		for _, existingUser := range users {
			if existingUser.GetEmail() == user.GetEmail() {
				return existingUser, nil
			}
		}
		users = append(users, user)
		return nil, nil
	},
	UpdateUser: func(user User) error {
		usersMutex.Lock()
		defer usersMutex.Unlock()
		for index, existingUser := range users {
			if existingUser.GetID() == user.GetID() {
				users[index] = user
				return nil
			}
		}
		return nil
	},
	LoadUserByVerificationID: func(id string) (User, error) {
		usersMutex.RLock()
		defer usersMutex.RUnlock()
		for _, user := range users {
			vid, _ := user.GetVerificationID()
			if vid == id {
				return user, nil
			}
		}
		return nil, nil
	},
	LoadUserByPasswordToken: func(token string) (User, error) {
		usersMutex.RLock()
		defer usersMutex.RUnlock()
		for _, user := range users {
			t, _ := user.GetPasswordToken()
			if t == token {
				return user, nil
			}
		}
		return nil, nil
	},
	LoadUserByEmail: func(email string) (User, error) {
		usersMutex.RLock()
		defer usersMutex.RUnlock()
		for _, user := range users {
			if user.GetEmail() == email {
				return user, nil
			}
		}
		return nil, nil
	},
	LoggedIn: nil,
	ThrottleVerification: func() {
		pauseMutex.Lock()
		time.Sleep(time.Second)
		pauseMutex.Unlock()
	},
	ThrottleLogin: func(email string) {
		userMutexesMutex.Lock()
		if len(userMutexes) >= 1000 {
			userMutexes = make(map[string]*sync.Mutex)
		}
		mutex, ok := userMutexes[email]
		if !ok {
			mutex = &sync.Mutex{}
			userMutexes[email] = mutex
		}
		userMutexesMutex.Unlock()
		mutex.Lock()
		time.Sleep(time.Second)
		mutex.Unlock()
	},
}
