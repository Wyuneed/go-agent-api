package entity

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
	UserRoleAgent UserRole = "agent"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Name         string
	Role         UserRole
	IsActive     bool
	Metadata     map[string]any
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewUser(email, passwordHash, name string) *User {
	return &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		Name:         name,
		Role:         UserRoleUser,
		IsActive:     true,
		Metadata:     make(map[string]any),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func (u *User) CanUseTools() bool {
	return u.IsActive && (u.Role == UserRoleAdmin || u.Role == UserRoleUser)
}

func (u *User) IsAdmin() bool {
	return u.Role == UserRoleAdmin
}
