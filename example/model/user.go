package model

type User struct {
	ID       int    `json:"id"       db:"id"`
	Email    int    `json:"email"    db:"email" validate:"email"`
	Password string `json:"password" db:"-"`
	Passhash string `json:"-"        db:"passhash"`
}
