package model

type Balance struct {
	ID        string `json:"id" db:"id"`
	UserID    string `json:"user_id" db:"user_id"`
	Current   int    `json:"current" db:"current"`
	Withdrawn string `json:"withdrawn" db:"withdrawn"`
}

type BalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type TransactionCreateRequest struct {
	Order       string `json:"order"`
	Sum         int64  `json:"sum"`
	UserID      string `json:"user_id"`
	ProcessedAt string `json:"processed_at"`
}
