package users

import (
	"time"

	"github.com/rivo/sessions"
)

// The states a user account is in at any given time.
const (
	StateCreated  = iota // User account has been created but not yet verified.
	StateVerified        // User account has been verified and can be used.
	StateExpired         // User account has expired. User can log in but don't have access to functionality anymore.
)

// User represents one user account. This is an extension of the sessions.User
// interface with additional getters and setters for fields used within this
// package.
//
// See also Config.NewUser for a description on how to create a new user.
type User interface {
	sessions.User

	// We need to be able to copy user IDs.
	SetID(id interface{})

	// At any time, a user is in exactly one state: StateCreated, StateVerified,
	// or StateExpired.
	SetState(state int)
	GetState() int

	// The users email address. This package will always change email addresses to
	// lowercase before setting or comparing them.
	SetEmail(email string)
	GetEmail() string

	// A hash of the user's password. This package uses golang.org/x/crypto/bcrypt
	// to generate and compare hashes.
	SetPasswordHash(hash []byte)
	GetPasswordHash() []byte

	// Verification IDs (a 22 character long string and its creation time) are
	// used to verify new (or changed) user accounts.
	SetVerificationID(id string, created time.Time)
	GetVerificationID() (string, time.Time)

	// Password tokens (a 22 character long string and its creation time) are
	// used to reset forgotten passwords.
	SetPasswordToken(id string, created time.Time)
	GetPasswordToken() (string, time.Time)
}
