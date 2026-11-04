package user

// CreateUserRequest is what a client may send to sign up.
type CreateUserRequest struct {
	FirstName string `json:"firstName" validate:"required"`
	LastName  string `json:"lastName" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8,max=72"`
}

// PatchUserRequest holds the fields a user may change; omitted fields stay as they are.
type PatchUserRequest struct {
	FirstName *string `json:"firstName" validate:"omitempty,min=1"`
	LastName  *string `json:"lastName" validate:"omitempty,min=1"`
	Email     *string `json:"email" validate:"omitempty,email"`
}
