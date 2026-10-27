package auth

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=8,max=72"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// clientInfo is what the handler reads off the request for the service: IP, user agent and device_id cookie ("" if missing).
type clientInfo struct {
	ip, userAgent, deviceID string
}

// authTokens holds the values setAuthCookies writes to the client.
type authTokens struct {
	access, refresh, deviceID string
}
