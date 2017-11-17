package users

import (
	"net/http"
	"strings"
	"testing"

	"github.com/rivo/sessions"
)

func TestLogInPageLoggedOut(t *testing.T) {
	computed, _ := runRequest(nil, nil, nil, LogIn)
	assertString("HOLF", computed, t)
}

func TestLogInPageLoggedIn(t *testing.T) {
	computed, _ := runRequest(&MyUser{email: "X", state: StateVerified}, nil, nil, LogIn)
	assertString("redirect", computed, t)
}

func TestLogInNonexistingUser(t *testing.T) {
	Config.LoadUserByEmail = func(email string) (User, error) {
		return nil, nil
	}
	computed, _ := runRequest(nil, nil, map[string]string{
		"email":    "@",
		"password": "12345",
	}, LogIn)
	assertString("HOL!WL!F", computed, t)
}

func TestLogInWrongPassword(t *testing.T) {
	user := &MyUser{
		email:        "X",
		state:        StateVerified,
		passwordHash: []byte("12345"),
	}
	Config.LoadUserByEmail = func(id string) (User, error) {
		return user, nil
	}
	computed, _ := runRequest(nil, nil, map[string]string{
		"email":    "@",
		"password": "12345",
	}, LogIn)
	assertString("HOL!WL!F", computed, t)
}

func TestLogInWrongState(t *testing.T) {
	user := &MyUser{
		email:        "X",
		state:        StateCreated,
		passwordHash: []byte("$2a$10$bRkkfyQZRP3eQkgkRvoktuvt6.ebieDKr/hZY4zWHg98HHEhbTHCC"),
	}
	Config.LoadUserByEmail = func(id string) (User, error) {
		return user, nil
	}
	computed, _ := runRequest(nil, nil, map[string]string{
		"email":    "@",
		"password": "12345",
	}, LogIn)
	assertString("HOS!VI!EF", computed, t)
}

func TestLogIn(t *testing.T) {
	var event string
	user := &MyUser{
		email:        "X",
		state:        StateVerified,
		passwordHash: []byte("$2a$10$bRkkfyQZRP3eQkgkRvoktuvt6.ebieDKr/hZY4zWHg98HHEhbTHCC"),
	}
	Config.LoadUserByEmail = func(id string) (User, error) {
		return user, nil
	}
	Config.LoggedIn = func(user User, ipAddress string) {
		event = "logged in"
	}
	computed, _ := runRequest(nil, nil, map[string]string{
		"email":    "@",
		"password": "12345",
	}, LogIn)
	assertString("redirect", computed, t)
	assertString("logged in", event, t)
}

func TestLoggedInUnverifiedUser(t *testing.T) {
	computed, _ := runRequest(&MyUser{
		state: StateCreated,
	}, nil, nil, func(response http.ResponseWriter, request *http.Request) {
		user, session, out := IsLoggedIn(response, request)
		if user != nil {
			t.Error("A user is logged in")
		}
		if session == nil {
			t.Error("Did not receive a session")
		}
		if !out {
			t.Error("Nothing was output")
		}
	})
	assertString("HOS!VI!EF", computed, t)
}

func TestLoggedIn(t *testing.T) {
	computed, _ := runRequest(&MyUser{
		state: StateVerified,
	}, nil, nil, func(response http.ResponseWriter, request *http.Request) {
		user, session, out := IsLoggedIn(response, request)
		if user == nil {
			t.Error("No user is logged in")
		}
		if session == nil {
			t.Error("Did not receive a session")
		}
		if out {
			t.Error("Something was output")
		}
	})
	assertString("", computed, t)
}

// Also tests program errors.
func TestLogOutWithGet(t *testing.T) {
	computed, _ := runRequest(nil, nil, nil, LogOut)
	expected := "PELogout method must be POST"
	if !strings.HasPrefix(computed, expected) {
		t.Errorf(`Expected prefix "%s" but string is "%s"`, expected, computed)
	}
}

func TestLogOutLoggedOut(t *testing.T) {
	computed, _ := runRequest(nil, nil, map[string]string{}, LogOut)
	assertString("redirect", computed, t)
}

func TestLogOut(t *testing.T) {
	computed, _ := runRequest(&MyUser{
		state: StateVerified,
	}, nil, map[string]string{}, func(response http.ResponseWriter, request *http.Request) {
		LogOut(response, request)
		sessions.Persistence = sessions.ExtendablePersistenceLayer{
			LoadSessionFunc: func(id string) (*sessions.Session, error) {
				// The session ID has changed so a new load is triggered. This session
				return nil, nil
			},
		}
		user, _, _ := IsLoggedIn(response, request)
		if user != nil {
			t.Error("User is still logged in")
		}
	})
	assertString("redirect", computed, t)
}
