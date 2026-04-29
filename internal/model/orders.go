package model

type OrderCreateRequest struct {
	UserID      string `json:"user_id"`
	OrderNumber string `json:"order_number"`
	UploadedAt  string `json:"uploaded_at"`
}

type Order struct {
	ID          string `json:"id" db:"id"`
	UserID      string `json:"user_id" db:"user_id"`
	Accrual     int    `json:"accrual" db:"accrual"`
	Status      string `json:"status" db:"status"`
	OrderNumber string `json:"order_number" db:"order_number"`
	UploadedAt  string `json:"uploaded_at" db:"uploaded_at"`
}
