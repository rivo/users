package users

import (
	"testing"
	"time"
)

func TestForgottenPasswordPageLoggedIn(t *testing.T) {
	computed, _ := runRequest(&MyUser{email: "X", state: StateVerified}, nil, nil, ForgottenPassword)
	assertString("redirect", computed, t)
}

func TestForgottenPasswordPageLoggedOut(t *testing.T) {
	computed, _ := runRequest(nil, nil, nil, ForgottenPassword)
	assertString("HOFPF", computed, t)
}

func TestForgottenPasswordExistingUser(t *testing.T) {
	user := &MyUser{email: "@", state: StateVerified}
	Config.LoadUserByEmail = func(email string) (User, error) {
		return user, nil
	}
	html, mail := runRequest(nil, nil, map[string]string{
		"email": "@",
	}, ForgottenPassword)
	assertString("HOLS@F", html, t)
	assertString("RE", mail, t)
	if user.passwordToken == "" {
		t.Error("No password token set")
	}
}

func TestForgottenPasswordUnknownUser(t *testing.T) {
	Config.LoadUserByEmail = func(email string) (User, error) {
		return nil, nil
	}
	html, mail := runRequest(nil, nil, map[string]string{
		"email": "@",
	}, ForgottenPassword)
	assertString("HOLS@F", html, t)
	assertString("RU", mail, t)
}

func TestResetPasswordUnknownUser(t *testing.T) {
	Config.LoadUserByPasswordToken = func(token string) (User, error) {
		return nil, nil
	}
	computed, _ := runRequest(nil, nil, nil, ResetPassword)
	assertString("HOFP!TNF!F", computed, t)
}

func TestResetPasswordExpiredToken(t *testing.T) {
	Config.LoadUserByPasswordToken = func(token string) (User, error) {
		return &MyUser{passwordToken: "12345", tokenCreated: time.Now().Add(-48 * time.Hour), state: StateVerified}, nil
	}
	computed, _ := runRequest(nil, map[string]string{"token": "12345"}, nil, ResetPassword)
	assertString("HOFP!TE!F", computed, t)
}

func TestResetPasswordPage(t *testing.T) {
	Config.LoadUserByPasswordToken = func(token string) (User, error) {
		return &MyUser{passwordToken: "12345", tokenCreated: time.Now(), state: StateVerified}, nil
	}
	computed, _ := runRequest(nil, map[string]string{"token": "12345"}, nil, ResetPassword)
	assertString("HORP12345F", computed, t)
}

func TestResetPasswordNoMatchingPasswords(t *testing.T) {
	Config.LoadUserByPasswordToken = func(token string) (User, error) {
		return &MyUser{passwordToken: "12345", tokenCreated: time.Now(), state: StateVerified}, nil
	}
	computed, _ := runRequest(nil, nil, map[string]string{
		"token":           "12345",
		"password":        "abcd",
		"passwordconfirm": "abc",
	}, ResetPassword)
	assertString("HORP12345!2!F", computed, t)
}

func TestResetPasswordInvalidPassword(t *testing.T) {
	Config.LoadUserByPasswordToken = func(token string) (User, error) {
		return &MyUser{passwordToken: "12345", tokenCreated: time.Now(), state: StateVerified}, nil
	}
	computed, _ := runRequest(nil, nil, map[string]string{
		"token":           "12345",
		"password":        "abc",
		"passwordconfirm": "abc",
	}, ResetPassword)
	assertString("HORP12345!a3!F", computed, t)
}

func TestResetPassword(t *testing.T) {
	user := &MyUser{passwordToken: "12345", tokenCreated: time.Now(), state: StateVerified}
	Config.LoadUserByPasswordToken = func(token string) (User, error) {
		return user, nil
	}
	computed, _ := runRequest(nil, nil, map[string]string{
		"token":           "12345",
		"password":        "kjhvasfiuwbucj",
		"passwordconfirm": "kjhvasfiuwbucj",
	}, ResetPassword)
	assertString("HOPRF", computed, t)
	if len(user.passwordHash) == 0 {
		t.Error("No password was set")
	}
}
