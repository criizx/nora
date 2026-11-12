package protocol

type TunnelInfo struct {
	ID       string `json:"id"`
	Protocol string `json:"protocol"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

type AuthRequest struct {
	Token    string       `json:"token"`
	Username string       `json:"username"`
	Tunnels  []TunnelInfo `json:"tunnels"`
}

type AuthResponse struct {
	Success bool              `json:"success"`
	Error   string            `json:"error,omitempty"`
	Urls    map[string]string `json:"urls"`
}

type StatusUpdate struct {
	Type     string            `json:"type"` //always "status"
	Statuses map[string]string `json:"urls"`
}
