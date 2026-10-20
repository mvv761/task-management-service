package dto

type RegisterRequest struct {
	Email    string `json:"email" binding:"required" example:"alex@mail.test"`
	Password string `json:"password" binding:"required" example:"qwerty123"`
	Name     string `json:"name" binding:"required" example:"Alex Petrov"`
}

type RegisterResponse struct {
	ID    uint64 `json:"id" example:"1"`
	Email string `json:"email" example:"alex@mail.test"`
	Name  string `json:"name" example:"Alex Petrov"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required" example:"alex@mail.test"`
	Password string `json:"password" binding:"required" example:"qwerty123"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}
