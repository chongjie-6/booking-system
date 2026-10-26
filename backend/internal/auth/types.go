package auth

import "time"

type RefreshTokenRow struct {
	TokenHash string    `db:"token_hash"`
	UserID    string    `db:"user_id"`
	UserRole  string    `db:"user_role"`
	ExpiresAt time.Time `db:"expires_at"`
	IsRevoked bool      `db:"is_revoked"`
	IPAddress *string   `db:"ip_address"`
	UserAgent *string   `db:"user_agent"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=8,max=72"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}
