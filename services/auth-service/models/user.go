package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

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

func (u *User) HashPassword() error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}
