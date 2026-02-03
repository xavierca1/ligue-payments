package whatsapp


type SendMessageInput struct {
	PhoneNumber  string   // Ex: "5511999999999"
	TemplateName string   // Ex: "welcome_notification"
	Parameters   []string // Ex: []string{"João Silva", "Plano Premium"}
}


type SendMessageResponse struct {
	MessageID string `json:"messages"`
	Contacts  []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts"`
	Error *ErrorResponse `json:"error"`
}


type ErrorResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Type    string `json:"type"`
}
