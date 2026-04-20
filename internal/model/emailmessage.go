package model

type EmailMessage struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Value    int    `json:"value"`
}
