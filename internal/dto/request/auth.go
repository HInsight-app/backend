package dto

type RegisterRequest struct {
	Email       string `json:"email" validate:"required,email"`
	DisplayName string `json:"display_name" validate:"required,min=2"`
	Password    string `json:"password" validate:"required,min=8"`
	Remember    bool   `json:"remember"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	Remember bool   `json:"remember"`
}
