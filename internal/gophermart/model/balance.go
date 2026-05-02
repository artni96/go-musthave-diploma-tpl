package model

type Balance struct {
	ID        string `json:"id" db:"id"`
	UserID    string `json:"user_id" db:"user_id"`
	Current   int    `json:"current" db:"current"`
	Withdrawn string `json:"withdrawn" db:"withdrawn"`
}
