package doc24

import (
	"strings"

	"github.com/xavierca1/ligue-payments/internal/entity"
)

func mapToDoc24(c *entity.Customer) CreateBeneficiaryInput {
	// 1. Quebra o nome completo em Nome + Sobrenome
	parts := strings.SplitN(c.Name, " ", 2)
	sobrenome := ""
	if len(parts) > 1 {
		sobrenome = parts[1]
	} else {
		sobrenome = parts[0] // Fallback se o cara só tiver um nome
	}

	// 2. Traduz o Gênero
	genero := "O" // Outros/Default
	if c.Gender == 1 {
		genero = "M"
	} // Ajuste conforme sua regra de negócio
	if c.Gender == 2 {
		genero = "F"
	}

	// 3. Formata Data (Assume que no banco já está YYYY-MM-DD ou formata aqui)
	// Se c.BirthDate for string "YYYY-MM-DD", passa direto.

	return CreateBeneficiaryInput{
		Nombre:               parts[0],
		Apellido:             sobrenome,
		Genero:               genero,
		IdTipoIdentificacion: 1,     // ⚠️ Atenção: Confirmar se 1 é CPF na API deles
		ValorIdentificacion:  c.CPF, // Remove pontos e traços se a API pedir limpo
		FechaNacimiento:      c.BirthDate,
		Email:                c.Email,
		Telefono:             c.Phone,
		IdPais:               "BR",
		BrandID:              c.ID, // Seu UUID
		Guest:                1,
	}
}
