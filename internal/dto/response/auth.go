package dto

type AuthResponse struct {
	User  UserResponse `json:"user"`
	Token string       `json:"token"` // This is the RAW token sent to the client
}

type UserResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}
