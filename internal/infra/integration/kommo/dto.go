package kommo


type SendWhatsAppInput struct {
	PhoneNumber string // Ex: "5511999999999"
	Name        string // Nome do cliente
	PlanName    string // Nome do plano
	Message     string // Mensagem customizada (opcional)
}


type ContactResponse struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}
