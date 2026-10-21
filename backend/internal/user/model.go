package user

import "time"

type User struct {
	UserID         string    `db:"user_id" json:"userID"`
	UserRole       string    `db:"user_role" json:"userRole"`
	HashedPassword *string   `db:"hashed_password" json:"-"`
	FirstName      string    `db:"first_name" json:"firstName"`
	LastName       string    `db:"last_name" json:"lastName"`
	Email          string    `db:"email" json:"email"`
	CreatedAt      time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time `db:"updated_at" json:"updatedAt"`
}
