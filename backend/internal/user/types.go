package user

// CreateUserRequest is what a client may send to sign up.
type CreateUserRequest struct {
	FirstName string `json:"firstName" binding:"required"`
	LastName  string `json:"lastName" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8,max=72"`
}

// PatchUserRequest holds the fields a user may change; omitted fields stay as they are.
type PatchUserRequest struct {
	FirstName *string `json:"firstName" binding:"omitempty,min=1"`
	LastName  *string `json:"lastName" binding:"omitempty,min=1"`
	Email     *string `json:"email" binding:"omitempty,email"`
}
