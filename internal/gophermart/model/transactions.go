package model

type Transaction struct {
	Order  string `json:"order"`
	Sum    int64  `json:"sum"`
	UserID string `json:"user_id"`
}

//type TransactionCreateRequest struct {
//	Order       string `json:"order"`
//	Sum         int64  `json:"sum"`
//	UserID      string `json:"user_id"`
//	ProcessedAt string `json:"processed_at"`
//}
