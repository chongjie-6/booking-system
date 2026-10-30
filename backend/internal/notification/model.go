package notification

// event is a message on the notifications topic. Type picks the notification; the other fields are its details.
type event struct {
	Type      string `json:"type"`
	UserID    string `json:"userID"`
	IP        string `json:"ip"`
	UserAgent string `json:"userAgent"`
}
