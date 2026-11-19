package dto

// GetCentrifugoTokenInput for requesting a Centrifugo connection JWT
type GetCentrifugoTokenInput struct{}

type CentrifugoTokenResponse struct {
	Token     string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." doc:"Centrifugo connection JWT"`
	ExpiresAt int64  `json:"expires_at" example:"1700000000"`
	WSURL     string `json:"ws_url" example:"wss://chat.techguild.io/connection/websocket"`
}

type GetCentrifugoTokenOutput struct {
	Body CentrifugoTokenResponse
}

// CentrifugoSubscribeProxyRequest is the payload sent by Centrifugo when a client subscribes
type CentrifugoSubscribeProxyRequest struct {
	Client    string `json:"client,omitempty"`
	Transport string `json:"transport,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
	Encoding  string `json:"encoding,omitempty"`
	User      string `json:"user" required:"true" doc:"User ID attempting subscription"`
	Channel   string `json:"channel" required:"true" doc:"Target channel name or ID"`
}

type CentrifugoSubscribeProxyInput struct {
	Body CentrifugoSubscribeProxyRequest
}

type CentrifugoProxyResult struct {
	// Optional Centrifugo subscription options
	ExpireAt *int64 `json:"expire_at,omitempty"`
	Info     []byte `json:"info,omitempty"`
}

type CentrifugoProxyError struct {
	Code    uint32 `json:"code"`
	Message string `json:"message"`
}

type CentrifugoSubscribeProxyResponse struct {
	Result *CentrifugoProxyResult `json:"result,omitempty"`
	Error  *CentrifugoProxyError  `json:"error,omitempty"`
}

type CentrifugoSubscribeProxyOutput struct {
	Body CentrifugoSubscribeProxyResponse
}

