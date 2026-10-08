package handler

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Detail  string `json:"detail,omitempty"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type FirebaseStatusResponse struct {
	Connected bool   `json:"connected"`
	Message   string `json:"message"`
	Error     string `json:"error,omitempty"`
}

type ProductDTO struct {
	ID      string `json:"id"`
	Charges int64  `json:"charges"`
	Price   string `json:"price"`
}

type ProductsResponse struct {
	Products []ProductDTO `json:"products"`
}

type RelayerStatusResponse struct {
	Address     string `json:"address"`
	Online      bool   `json:"online"`
	BalanceNano int64  `json:"balanceNano,omitempty"`
	Seqno       int    `json:"seqno,omitempty"`
	Deployed    bool   `json:"deployed,omitempty"`
	Error       string `json:"error,omitempty"`
}

type PayloadResponse struct {
	Payload string `json:"payload"`
}

type TonProofCheckResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expiresAt"`
	UserID    string `json:"userId"`
	PublicKey string `json:"publicKey"`
	Address   string `json:"address"`
	NewUser   bool   `json:"newUser"`
}

type MeResponse struct {
	UserID    string `json:"userId"`
	PublicKey string `json:"publicKey"`
}

type BalanceResponse struct {
	UserID    string `json:"userId"`
	Balance   int64  `json:"balance"`
	Reserved  int64  `json:"reserved"`
	Available int64  `json:"available"`
}

type EntryDTO struct {
	ID            string `json:"id"`
	Op            string `json:"op"`
	Amount        int64  `json:"amount"`
	BalanceAfter  int64  `json:"balanceAfter"`
	ReservedAfter int64  `json:"reservedAfter"`
	Reason        string `json:"reason"`
	CreatedAt     string `json:"createdAt"`
}

type TransactionsResponse struct {
	UserID       string     `json:"userId"`
	Transactions []EntryDTO `json:"transactions"`
}

type DecisionResponse struct {
	Supported    bool   `json:"supported"`
	Allowed      bool   `json:"allowed"`
	RejectReason string `json:"rejectReason"`
	Charge       int64  `json:"charge"`
}

type MessageSentResponse struct {
	Status string `json:"status"`
	TxHash string `json:"txHash"`
	Charge int64  `json:"charge"`
}

type PrintQuoteResponse struct {
	QuoteID   string `json:"quoteId"`
	BufferTON string `json:"bufferTON"`
	Charge    int64  `json:"charge"`
	ExpiresAt int64  `json:"expiresAt"`
}

type PrintExecuteResponse struct {
	QuoteID string `json:"quoteId"`
	Status  string `json:"status"`
	TxHash  string `json:"txHash"`
	Charge  int64  `json:"charge"`
}

type PrintStatusResponse struct {
	QuoteID   string `json:"quoteId"`
	Status    string `json:"status"`
	ToAddress string `json:"toAddress"`
	BufferTON string `json:"bufferTON"`
	Charge    int64  `json:"charge"`
	TxHash    string `json:"txHash"`
}

type AdminCreditResponse struct {
	UserID     string `json:"userId"`
	Balance    int64  `json:"balance"`
	Reserved   int64  `json:"reserved"`
	Available  int64  `json:"available"`
	Idempotent bool   `json:"idempotent"`
	EntryID    string `json:"entryId"`
}

type KillswitchResponse struct {
	Paused bool `json:"paused"`
}

type ReconcileResponse struct {
	UserID           string `json:"userId"`
	AccountBalance   int64  `json:"accountBalance"`
	AccountReserved  int64  `json:"accountReserved"`
	ComputedBalance  int64  `json:"computedBalance"`
	ComputedReserved int64  `json:"computedReserved"`
	Consistent       bool   `json:"consistent"`
}

type EmulateResponse struct {
	Success       bool     `json:"success"`
	TotalFeesNano int64    `json:"totalFeesNano"`
	OutMessages   int      `json:"outMessages"`
	Destinations  []string `json:"destinations"`
}

type RelayerDevSendResponse struct {
	TxHash   string `json:"txHash"`
	From     string `json:"from"`
	To       string `json:"to"`
	Amount   string `json:"amount"`
	Explorer string `json:"explorer"`
}
