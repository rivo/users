package users

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/rivo/sessions"
)

var (
	// The users in our system.
	users      []User
	usersMutex sync.RWMutex
)

// Initialize this package.
func init() {
	// See Config user functions for more information.
	persistence, ok := sessions.Persistence.(sessions.ExtendablePersistenceLayer)
	if ok {
		persistence.LoadUserFunc = func(id interface{}) (sessions.User, error) {
			userID, ok := id.(string)
			if !ok {
				return nil, fmt.Errorf("User ID is not a string: %#v", id)
			}
			usersMutex.RLock()
			defer usersMutex.RUnlock()
			for _, user := range users {
				if user.GetID() == userID {
					return user, nil
				}
			}
			return nil, fmt.Errorf("User not found: %s", userID)
		}
	}
}

// Main makes your life simple by starting an HTTP server for you with the
// routes found in the Config variable. If you use this, for all remaining
// pages of your application, you only need to add your own handlers to the
// DefaultServerMux prior to calling this function. See package documentation
// for an example.
func Main() error {
	Config.Log.Printf("Starting HTTP server on %s", Config.ServerAddr)
	defer Config.Log.Printf("Stopping HTTP server")
	http.HandleFunc(Config.RouteSignUp, SignUp)
	http.HandleFunc(Config.RouteVerify, Verify)
	http.HandleFunc(Config.RouteLogIn, LogIn)
	http.HandleFunc(Config.RouteLogOut, LogOut)
	http.HandleFunc(Config.RouteForgottenPassword, ForgottenPassword)
	http.HandleFunc(Config.RouteResetPassword, ResetPassword)
	http.HandleFunc(Config.RouteChange, Change)

	return http.ListenAndServe(Config.ServerAddr, nil)
}
