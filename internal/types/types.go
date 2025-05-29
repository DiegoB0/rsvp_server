package types

import "time"

type User struct {
	ID        int       `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     string    `json:"emal"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"CreatedAt"`
}

type RegisterUserPayload struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"emal"`
	Password  string `json:"password"`
}

type LoginUserPayload struct {
	Email    string
	Password string
}
