package doc24

type CreateBeneficiaryInput struct {
	Nombre               string `json:"nombre"`                    // Obrigatório
	Apellido             string `json:"apellido"`                  // Obrigatório
	Genero               string `json:"genero"`                    // M, F
	IdTipoIdentificacion int    `json:"id_tipo_de_identificacion"` // Dica: Tente 1 ou procure na doc de "tipos"
	ValorIdentificacion  string `json:"valor_identificacion"`      // O CPF
	FechaNacimiento      string `json:"fecha_de_nacimiento"`       // YYYY-MM-DD
	Email                string `json:"email"`
	Telefono             string `json:"telefono,omitempty"`
	IdPais               string `json:"id_pais"`  // "BR"
	BrandID              string `json:"brand_id"` // O ID do cliente no SEU banco (pra rastreio)
	Guest                int    `json:"guest"`    // Manda 1
}

type CreateBeneficiaryOutput struct {
	ID     int    `json:"id"`
	Status string `json:"status"` // Geralmente não vem explícito, mas valida pelo HTTP 200/201
}
