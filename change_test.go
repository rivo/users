package users

import (
	"strings"
	"testing"
)

func TestChangePageLoggedOut(t *testing.T) {
	computed, _ := runRequest(nil, nil, nil, Change)
	expected := "PEThis page may only be accessed when you are logged in"
	if !strings.HasPrefix(computed, expected) {
		t.Errorf(`Expected prefix "%s" but string is "%s"`, expected, computed)
	}
}

func TestChangePageLoggedIn(t *testing.T) {
	computed, _ := runRequest(&MyUser{email: "x", state: StateVerified}, nil, nil, Change)
	assertString("HICExF", computed, t)
}

func TestChangeNoData(t *testing.T) {
	computed, _ := runRequest(&MyUser{email: "x", state: StateVerified}, nil, map[string]string{
		"email": "x",
	}, Change)
	assertString("HICExF", computed, t)
}

func TestChangeNoCurrentPassword(t *testing.T) {
	computed, _ := runRequest(&MyUser{
		email: "x",
		state: StateVerified,
	}, nil, map[string]string{
		"email": "y",
	}, Change)
	assertString("HIC!NCP!EyF", computed, t)
}

func TestChangeWrongCurrentPassword(t *testing.T) {
	computed, _ := runRequest(&MyUser{
		email:        "x",
		state:        StateVerified,
		passwordHash: []byte("$2a$10$bRkkfyQZRP3eQkgkRvoktuvt6.ebieDKr/hZY4zWHg98HHEhbTHCC"),
	}, nil, map[string]string{
		"email":           "y",
		"currentpassword": "12345?",
	}, Change)
	assertString("HIC!WCP!EyF", computed, t)
}

func TestChangePasswordsDontMatch(t *testing.T) {
	computed, _ := runRequest(&MyUser{
		email:        "x",
		state:        StateVerified,
		passwordHash: []byte("$2a$10$bRkkfyQZRP3eQkgkRvoktuvt6.ebieDKr/hZY4zWHg98HHEhbTHCC"),
	}, nil, map[string]string{
		"email":           "x",
		"currentpassword": "12345",
		"password":        "abc",
		"passwordconfirm": "abcd",
	}, Change)
	assertString("HIC!2!ExF", computed, t)
}

func TestChangeInvalidPassword(t *testing.T) {
	computed, _ := runRequest(&MyUser{
		email:        "x",
		state:        StateVerified,
		passwordHash: []byte("$2a$10$bRkkfyQZRP3eQkgkRvoktuvt6.ebieDKr/hZY4zWHg98HHEhbTHCC"),
	}, nil, map[string]string{
		"email":           "x",
		"currentpassword": "12345",
		"password":        "abc",
		"passwordconfirm": "abc",
	}, Change)
	assertString("HIC!a3!ExF", computed, t)
}

func TestChangePasswordChange(t *testing.T) {
	hash := []byte("$2a$10$bRkkfyQZRP3eQkgkRvoktuvt6.ebieDKr/hZY4zWHg98HHEhbTHCC")
	user := &MyUser{
		email:        "x",
		state:        StateVerified,
		passwordHash: hash,
	}
	computed, _ := runRequest(user, nil, map[string]string{
		"email":           "x",
		"currentpassword": "12345",
		"password":        "lkasflkhasf",
		"passwordconfirm": "lkasflkhasf",
	}, Change)
	assertString("HIICF", computed, t)
	if string(hash) == string(user.passwordHash) {
		t.Error("Password hashes are still the same")
	}
}

func TestChangeInvalidEmail(t *testing.T) {
	computed, _ := runRequest(&MyUser{
		email:        "x",
		state:        StateVerified,
		passwordHash: []byte("$2a$10$bRkkfyQZRP3eQkgkRvoktuvt6.ebieDKr/hZY4zWHg98HHEhbTHCC"),
	}, nil, map[string]string{
		"email":           "y",
		"currentpassword": "12345",
	}, Change)
	assertString("HIC!01!EyF", computed, t)
}

func TestChangeEmailExists(t *testing.T) {
	user := &MyUser{
		email:        "x",
		state:        StateVerified,
		passwordHash: []byte("$2a$10$bRkkfyQZRP3eQkgkRvoktuvt6.ebieDKr/hZY4zWHg98HHEhbTHCC"),
	}
	Config.LoadUserByEmail = func(email string) (User, error) {
		return user, nil
	}
	html, mail := runRequest(user, nil, map[string]string{
		"email":           "@",
		"currentpassword": "12345",
	}, Change)
	assertString("HOICF", html, t)
	assertString("VE", mail, t)
}

func TestChangeEmail(t *testing.T) {
	user := &MyUser{
		email:        "x",
		state:        StateVerified,
		passwordHash: []byte("$2a$10$bRkkfyQZRP3eQkgkRvoktuvt6.ebieDKr/hZY4zWHg98HHEhbTHCC"),
	}
	Config.LoadUserByEmail = func(email string) (User, error) {
		return nil, nil
	}
	html, mail := runRequest(user, nil, map[string]string{
		"email":           "@",
		"currentpassword": "12345",
	}, Change)
	assertString("HOICF", html, t)
	assertString("VC", mail, t)
}
