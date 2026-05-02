package model

type OrderCreateRequest struct {
	UserID     string `json:"user_id"`
	Number     string `json:"number"`
	UploadedAt string `json:"uploaded_at"`
}

type Order struct {
	ID         string `json:"id" db:"id"`
	UserID     string `json:"user_id" db:"user_id"`
	Accrual    int    `json:"accrual" db:"accrual"`
	Status     string `json:"status" db:"status"`
	Number     string `json:"number" db:"number"`
	UploadedAt string `json:"uploaded_at" db:"uploaded_at"`
}

type OrderUpdateRequest struct {
	Number  string `json:"number" db:"number"`
	Accrual int    `json:"accrual" db:"accrual"`
	Status  string `json:"status" db:"status"`
	UserID  string `json:"user_id" db:"user_id"`
}

type OrderStatusUpdateRequest struct {
	Number string `json:"number"`
	Status string `json:"status"`
}
