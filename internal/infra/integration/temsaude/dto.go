package temsaude

type TemAdesaoRequest struct {
	// Campos de Identificação
	CpfTitular     string `json:"cpfTitular"` // Geralmente igual ao CPF para titular
	Cpf            string `json:"cpf"`
	Nome           string `json:"Nome"`
	NomeSocial     string `json:"nome_social,omitempty"` // Opcional
	DataNascimento string `json:"data_nascimento"`       // YYYY-MM-DD
	Sexo           int    `json:"Sexo"`                  // Inteiro!
	Email          string `json:"email"`
	Telefone       string `json:"Telefone"`
	IdentExterno   string `json:"ident_externo"` // Seu ID interno (uuid)

	Logradouro     string `json:"Logradouro"`
	NumeroEndereco string `json:"NumeroEndereco"`
	Complemento    string `json:"Complemento"`
	Bairro         string `json:"Bairro"`
	Cidade         string `json:"Cidade"`
	Estado         string `json:"Estado"`
	CEP            string `json:"CEP"`

	CodOnix       int    `json:"CodOnix"` // Inteiro (7065)
	Cnpj          string `json:"cnpj"`
	NumeroCartao  int    `json:"NumeroCartao"`  // Inteiro (0)
	NumeroDaSorte int    `json:"numerodasorte"` // Inteiro (0)
	TokenZeus     string `json:"tokenzeus"`     // O Token de Auth vai aqui dentro
	CN            string `json:"cn"`            // Campo string genérico
}

type TemAdesaoResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	// Mapeie outros campos de retorno se houver
}
