package user

import (
	"time"

	"github.com/google/uuid"
)

type RegisterCustomerRequest struct {
	FirstName string `json:"firstName" binding:"required,max=255"`
	LastName  string `json:"lastName" binding:"required,max=255"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8,max=72"`
	Code      string `json:"code" binding:"required,len=6,numeric"`
}

type RegisterOwnerRequest struct {
	FirstName    string `json:"firstName" binding:"required,max=255"`
	LastName     string `json:"lastName" binding:"required,max=255"`
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=8,max=72"`
	Code         string `json:"code" binding:"required,len=6,numeric"`
	BusinessName string `json:"businessName" binding:"required,min=2,max=128"`
	VATNumber    string `json:"vatNumber" binding:"required,len=11,alphanum"`
}

type Response struct {
	ID       uuid.UUID    `json:"id"`
	Name     UserName     `json:"name"`
	Email    string       `json:"email"`
	Role     string       `json:"role"`
	Status   string       `json:"status"`
	Metadata UserMetadata `json:"metadata"`
}

type UserName struct {
	First string `json:"first"`
	Last  string `json:"last"`
}

type UserMetadata struct {
	LastLogin   *time.Time `json:"lastLogin"`
	MemberSince time.Time  `json:"memberSince"`
	LastUpdate  *time.Time `json:"lastUpdate"`
}
