// Package models defines the persistence entities of the auth service together
// with the password-hashing helpers that operate on them.
package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User is a Moda Style account.
//
// The JSON tags deliberately omit the password hash, the reset token and its
// expiry (they all use json:"-") so a serialised User can never leak credential
// material to a client. Role is a coarse-grained string ("user" or "admin")
// and Banned is the permanent-ban flag used by the admin service.
type User struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	Name             string         `gorm:"size:100;not null" json:"name"`
	Email            string         `gorm:"size:255;uniqueIndex;not null" json:"email"`
	Password         string         `gorm:"size:255;not null" json:"-"`
	Role             string         `gorm:"size:20;default:'user';not null" json:"role"`
	Banned           bool           `gorm:"default:false;not null" json:"banned"`
	ResetToken       *string        `gorm:"size:255" json:"-"`
	ResetTokenExpiry *time.Time     `json:"-"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// HashPassword replaces Password with its bcrypt hash produced at
// bcrypt.DefaultCost. It must be called before the first save and returns the
// error from bcrypt rather than storing a partially hashed value.
func (u *User) HashPassword() error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword reports whether password matches the stored bcrypt hash. It is
// the counterpart of HashPassword and never reveals the hash itself.
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// IsAdmin reports whether the account holds the "admin" role.
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}
