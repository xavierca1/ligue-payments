package asaas

type PixOutput struct {
	CopyPaste string
	URL       string
}

// Struct auxiliar para ler a lista de pagamentos do Asaas
type listPaymentsResponse struct {
	Data []struct {
		ID string `json:"id"` // O Payment ID que queremos!
	} `json:"data"`
}

// Resposta quando criamos a assinatura
type asaasSubscriptionResponse struct {
	ID string `json:"id"`
}
type SubscribePixInput struct {
	CustomerID string
	Price      int64
}

// Resposta quando listamos as cobranças (para achar o ID do pagamento)
type asaasListPaymentsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type asaasQrCodeResponse struct {
	EncodedImage string `json:"encodedImage"`
	Payload      string `json:"payload"`
}
type SubscribeInput struct {
	CustomerID string
	Price      float64

	// Dados do Cartão
	CardHolderName string
	CardNumber     string
	CardMonth      string
	CardYear       string
	CardCCV        string

	// Dados do Titular (Necessários para evitar erro 400)
	HolderEmail      string
	HolderCpfCnpj    string
	HolderPostalCode string
	HolderAddressNum string
	HolderPhone      string
}

type CreateCustomerInput struct {
	Name          string
	Email         string
	CpfCnpj       string
	Phone         string
	MobilePhone   string
	PostalCode    string
	AddressNumber string
}

type createCustomerRequest struct {
	Name                 string `json:"name"`
	Email                string `json:"email"`
	CpfCnpj              string `json:"cpfCnpj"`
	Phone                string `json:"phone"`
	MobilePhone          string `json:"mobilePhone"`
	PostalCode           string `json:"postalCode"`
	AddressNumber        string `json:"addressNumber"`
	NotificationDisabled bool   `json:"notificationDisabled"`
}

type customerResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// --- PAYLOADS: O que o Client manda para o Asaas (Interno) ---

// Request principal
type createSubscriptionRequest struct {
	Customer             string               `json:"customer"`
	BillingType          string               `json:"billingType"`
	Value                float64              `json:"value"`
	NextDueDate          string               `json:"nextDueDate"`
	Cycle                string               `json:"cycle"`
	Description          string               `json:"description"`
	CreditCard           creditCard           `json:"creditCard"`
	CreditCardHolderInfo creditCardHolderInfo `json:"creditCardHolderInfo"`
}

// Dados do cartão
type creditCard struct {
	HolderName  string `json:"holderName"`
	Number      string `json:"number"`
	ExpiryMonth string `json:"expiryMonth"`
	ExpiryYear  string `json:"expiryYear"`
	CCV         string `json:"ccv"`
}

// Dados do titular (Anti-fraude)
type creditCardHolderInfo struct {
	Name              string `json:"name"`
	Email             string `json:"email"`
	CpfCnpj           string `json:"cpfCnpj"`
	PostalCode        string `json:"postalCode"`
	AddressNumber     string `json:"addressNumber"`
	AddressComplement string `json:"addressComplement"`
	Phone             string `json:"phone"`
	MobilePhone       string `json:"mobilePhone"`
}

// --- RESPONSE: O que o Asaas devolve ---
type subscriptionResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}
type SubscribePix struct {
	CustomerID string
	PriceCents int64
}
