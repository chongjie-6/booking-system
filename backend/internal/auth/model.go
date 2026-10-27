package auth

import "time"

type RefreshTokenRow struct {
	TokenHash  string    `db:"token_hash"`
	UserID     string    `db:"user_id"`
	UserRole   string    `db:"user_role"`
	ExpiresAt  time.Time `db:"expires_at"`
	IsRevoked  bool      `db:"is_revoked"`
	IPAddress  *string   `db:"ip_address"`
	UserAgent  *string   `db:"user_agent"`
	DeviceHash *string   `db:"device_hash"`
}
