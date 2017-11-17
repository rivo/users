package users

import (
	"testing"
	"time"
)

func TestSignUpPageLoggedOut(t *testing.T) {
	computed, _ := runRequest(nil, nil, nil, SignUp)
	assertString("HOSEF", computed, t)
}

func TestSignUpPageLoggedIn(t *testing.T) {
	computed, _ := runRequest(&MyUser{email: "X", state: StateVerified}, nil, nil, SignUp)
	assertString("redirect", computed, t)
}

func TestSignUpNoEmail(t *testing.T) {
	computed, _ := runRequest(nil, nil, map[string]string{
		"email": "",
	}, SignUp)
	assertString("HOS!1!EF", computed, t)
}

func TestSignUpInvalidEmail(t *testing.T) {
	computed, _ := runRequest(nil, nil, map[string]string{
		"email": "x",
	}, SignUp)
	assertString("HOS!01!ExF", computed, t)
}

func TestSignUpPasswordsDontMatch(t *testing.T) {
	computed, _ := runRequest(nil, nil, map[string]string{
		"email":    "a@b",
		"password": "abc",
	}, SignUp)
	assertString("HOS!2!Ea@bF", computed, t)
}

func TestSignUpNoPassword(t *testing.T) {
	computed, _ := runRequest(nil, nil, map[string]string{
		"email": "a@b",
	}, SignUp)
	assertString("HOS!a3!Ea@bF", computed, t)
}

func TestSignUpAppPassword(t *testing.T) {
	computed, _ := runRequest(nil, nil, map[string]string{
		"email":           "test@example.com",
		"password":        "test@example.com",
		"passwordconfirm": "test@example.com",
	}, SignUp)
	assertString("HOS!b3!Etest@example.comF", computed, t)
}

func TestSignUpCompromisedPassword(t *testing.T) {
	computed, _ := runRequest(nil, nil, map[string]string{
		"email":           "a@b",
		"password":        "12345678",
		"passwordconfirm": "12345678",
	}, SignUp)
	assertString("HOS!c3!Ea@bF", computed, t)
}

func TestSignUpDictionaryPassword(t *testing.T) {
	computed, _ := runRequest(nil, nil, map[string]string{
		"email":           "a@b",
		"password":        "aardvarks",
		"passwordconfirm": "aardvarks",
	}, SignUp)
	assertString("HOS!d3!Ea@bF", computed, t)
}

func TestSignUpRepetitivePassword(t *testing.T) {
	computed, _ := runRequest(nil, nil, map[string]string{
		"email":           "a@b",
		"password":        "öööööööö",
		"passwordconfirm": "öööööööö",
	}, SignUp)
	assertString("HOS!e3!Ea@bF", computed, t)
}

func TestSignUpSequencePassword(t *testing.T) {
	computed, _ := runRequest(nil, nil, map[string]string{
		"email":           "a@b",
		"password":        "ertzuiop",
		"passwordconfirm": "ertzuiop",
	}, SignUp)
	assertString("HOS!f3!Ea@bF", computed, t)
}

func TestSignUpExistingAccount(t *testing.T) {
	backup := Config.SaveNewUserAtomic
	Config.SaveNewUserAtomic = func(user User) (User, error) {
		return &MyUser{
			email: "b@c",
			state: StateVerified,
		}, nil
	}
	html, mail := runRequest(nil, nil, map[string]string{
		"email":           "a@b",
		"password":        "lakjshfaksjhf",
		"passwordconfirm": "lakjshfaksjhf",
	}, SignUp)
	assertString("HOVSF", html, t)
	assertString("VE", mail, t)
	Config.SaveNewUserAtomic = backup
}

func TestSignUpExistingUnverifiedAccount(t *testing.T) {
	backup := Config.SaveNewUserAtomic
	Config.SaveNewUserAtomic = func(user User) (User, error) {
		return &MyUser{
			email: "b@c",
			state: StateCreated,
		}, nil
	}
	html, mail := runRequest(nil, nil, map[string]string{
		"email":           "a@b",
		"password":        "lakjshfaksjhf",
		"passwordconfirm": "lakjshfaksjhf",
	}, SignUp)
	assertString("HOVSF", html, t)
	assertString("VN", mail, t)
	Config.SaveNewUserAtomic = backup
}

func TestSignUp(t *testing.T) {
	html, mail := runRequest(nil, nil, map[string]string{
		"email":           "a@b",
		"password":        "lakjshfaksjhf",
		"passwordconfirm": "lakjshfaksjhf",
	}, SignUp)
	assertString("HOVSF", html, t)
	assertString("VN", mail, t)
}

func TestVerifyUnknownID(t *testing.T) {
	html, _ := runRequest(nil, map[string]string{
		"id": "12345",
	}, nil, Verify)
	assertString("HOS!VI!EF", html, t)
}

func TestVerifyExpiredID(t *testing.T) {
	Config.LoadUserByVerificationID = func(id string) (User, error) {
		return &MyUser{
			verificationID: "12345",
			vidCreated:     time.Now().Add(-365 * 24 * time.Hour),
		}, nil
	}
	html, _ := runRequest(nil, map[string]string{
		"id": "12345",
	}, nil, Verify)
	assertString("HOS!VE!EF", html, t)
}

func TestVerify(t *testing.T) {
	user := &MyUser{
		verificationID: "12345",
		vidCreated:     time.Now().Add(-time.Minute),
		state:          StateCreated,
	}
	Config.LoadUserByVerificationID = func(id string) (User, error) {
		return user, nil
	}
	html, _ := runRequest(nil, map[string]string{
		"id": "12345",
	}, nil, Verify)
	assertString("HOVF", html, t)
	if user.state != StateVerified {
		t.Error("User was not verified")
	}
}
