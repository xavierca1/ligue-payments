package usecase

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/asaas"
)

func NewCreateCustomerUseCase(
	repo CustomerRepositoryInterface,
	subRepo SubscriptionRepository,
	planRepo PlanRepositoryInterface,
	gateway PaymentGateway,
	queue QueueProducerInterface,
	emailService EmailService,
	welcomeBucketURL string,
) *CreateCustomerUseCase {
	return &CreateCustomerUseCase{
		Repo:             repo,
		SubRepo:          subRepo,
		PlanRepo:         planRepo,
		Gateway:          gateway,
		Queue:            queue,
		EmailService:     emailService,
		WelcomeBucketURL: welcomeBucketURL,
	}
}

// func (uc *CreateCustomerUseCase) Execute(ctx context.Context, input CreateCustomerInput) (*CreateCustomerOutput, error) {
// 	// 1. Validar e Buscar Dados Iniciais
// 	plan, err := uc.PlanRepo.FindByID(ctx, input.PlanID)
// 	if err != nil {
// 		return nil, fmt.Errorf("plano inválido: %w", err)
// 	}

// 	// 2. Montar Entidades em Memória (SEM SALVAR AINDA) 🧠
// 	// Geramos os IDs aqui no Go, assim temos controle total antes de ir pro banco
// 	customerID := uuid.New().String()

// 	genderInt, _ := strconv.Atoi(input.Gender)
// 	address := entity.Address{
// 		Street: input.Street, Number: input.Number, Complement: input.Complement,
// 		District: input.District, City: input.City, State: input.State, ZipCode: input.ZipCode,
// 	}

// 	// Criamos a struct manualmente para injetar o ID que acabamos de gerar
// 	customer := &entity.Customer{
// 		ID:        customerID,
// 		Name:      input.Name,
// 		Email:     input.Email,
// 		CPF:       input.CPF,
// 		Phone:     input.Phone,
// 		PlanID:    input.PlanID,
// 		ProductID: plan.ProductID,
// 		BirthDate: input.BirthDate, // Certifique-se que sua entity aceita string ou converta
// 		Gender:    genderInt,
// 		Address:   address,
// 		OnixCode:  input.OnixCode,
// 		CreatedAt: time.Now(),
// 		UpdatedAt: time.Now(),
// 	}

// 	asaasID, err := uc.Gateway.CreateCustomer(asaas.CreateCustomerInput{
// 		Name: input.Name, Email: input.Email, CpfCnpj: input.CPF,
// 		Phone: input.Phone, PostalCode: input.ZipCode,
// 		// Dica: Asaas pede externalReference, ajuda a lincar
// 		ExternalReference: input.ExternalReference,
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf("Asaas recusou o cliente: %w", err)
// 	}
// 	customer.GatewayID = asaasID

// 	// 4. Processar Pagamento no Asaas
// 	var pixData *asaas.PixOutput
// 	var status string = "WAITING_PAYMENT"

// 	// Tenta cobrar
// 	if input.PaymentMethod == "PIX" {
// 		_, pixData, err = uc.Gateway.SubscribePix(asaas.SubscribePixInput{
// 			CustomerID: asaasID,
// 			Price:      int64(plan.PriceCents),
// 		})
// 	} else {
// 		_, _, err = uc.Gateway.Subscribe(asaas.SubscribeInput{
// 			CustomerID:       asaasID,
// 			Price:            float64(plan.PriceCents) / 100.0,
// 			CardNumber:       input.CardNumber,
// 			CardHolderName:   input.CardHolder,
// 			CardMonth:        input.CardMonth,
// 			CardYear:         input.CardYear,
// 			CardCCV:          input.CardCVV,
// 			HolderEmail:      input.Email,
// 			HolderCpfCnpj:    input.CPF,
// 			HolderPostalCode: input.ZipCode,
// 			HolderAddressNum: input.Number,
// 			HolderPhone:      input.Phone,
// 		})
// 	}

// 	if err != nil {
// 		// Se deu erro no pagamento, retornamos o erro.
// 		// Como não salvamos nada no banco, NÃO PRECISA DE ROLLBACK! 🎉
// 		return nil, fmt.Errorf("Asaas recusou o pagamento (400): %w", err)
// 	}

// 	// 5. SUCESSO NO ASAAS! Agora sim, salvamos tudo no Banco. 💾

// 	// A) Salva Cliente
// 	if err := uc.Repo.Create(ctx, customer); err != nil {
// 		// Aqui seria o único lugar onde precisaríamos de rollback (cancelar no asaas),
// 		// mas erro de banco local é raríssimo comparado a erro de API.
// 		return nil, fmt.Errorf("erro crítico ao salvar cliente no banco: %w", err)
// 	}

// 	// B) Salva Assinatura
// 	subscription := &entity.Subscription{
// 		ID:              uuid.New().String(),
// 		CustomerID:      customer.ID,
// 		PlanID:          plan.ID,
// 		ProductID:       plan.ProductID,
// 		Amount:          plan.PriceCents,
// 		Status:          "PENDING",
// 		NextBillingDate: time.Now().AddDate(0, 1, 0),
// 		CreatedAt:       time.Now(),
// 		UpdatedAt:       time.Now(),
// 	}

// 	if err := uc.SubRepo.Create(ctx, subscription); err != nil {
// 		return nil, fmt.Errorf("erro crítico ao salvar assinatura no banco: %w", err)
// 	}

// 	// 6. Retorna Sucesso
// 	var pixCode, pixUrl string
// 	if pixData != nil {
// 		pixCode = pixData.CopyPaste
// 		pixUrl = pixData.URL
// 	}

// 	return &CreateCustomerOutput{
// 		ID:           customer.ID,
// 		Status:       status,
// 		PixCode:      pixCode,
// 		PixQRCodeURL: pixUrl,
// 		Msg:          "Pré-cadastro realizado com sucesso!",
// 	}, nil
// }

func (uc *CreateCustomerUseCase) Execute(ctx context.Context, input CreateCustomerInput) (*CreateCustomerOutput, error) {
	// 1. Validar e Buscar Dados Iniciais
	plan, err := uc.PlanRepo.FindByID(ctx, input.PlanID)
	if err != nil {
		return nil, fmt.Errorf("plano inválido: %w", err)
	}

	// 2. Montar Entidades em Memória (SEM SALVAR AINDA) 🧠
	// Geramos os IDs aqui no Go
	customerID := uuid.New().String()

	// Tratamento seguro do Gênero (string -> int)
	genderInt, _ := strconv.Atoi(input.Gender)
	if genderInt == 0 {
		genderInt = 1 // Fallback seguro
	}

	address := entity.Address{
		Street:     input.Street,
		Number:     input.Number,
		Complement: input.Complement,
		District:   input.District,
		City:       input.City,
		State:      input.State,
		ZipCode:    input.ZipCode,
	}

	// Criamos a struct
	customer := &entity.Customer{
		ID:        customerID,
		Name:      input.Name,
		Email:     input.Email,
		CPF:       input.CPF,
		Phone:     input.Phone,
		PlanID:    input.PlanID,
		ProductID: plan.ProductID, // 👈 Importante para evitar erro de FK
		BirthDate: input.BirthDate,
		Gender:    genderInt,
		Address:   address,
		OnixCode:  input.OnixCode,

		// 🆕 Mapeamento dos Termos (Vem do Front)
		TermsAccepted:   input.TermsAccepted,
		TermsAcceptedAt: parseDateOrNow(input.TermsAcceptedAt), // Helper simples ou time.Now()
		TermsVersion:    input.TermsVersion,

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 3. Criar Cliente no Asaas
	// ⚠️ AQUI ESTÁ O PULO DO GATO: Passamos o customerID como ExternalReference
	asaasID, err := uc.Gateway.CreateCustomer(asaas.CreateCustomerInput{
		Name:              input.Name,
		Email:             input.Email,
		CpfCnpj:           input.CPF,
		Phone:             input.Phone,
		PostalCode:        input.ZipCode,
		AddressNumber:     input.Number,
		ExternalReference: customerID, // <--- O ID do nosso banco vai pro Asaas aqui
	})
	if err != nil {
		return nil, fmt.Errorf("Asaas recusou o cliente: %w", err)
	}
	customer.GatewayID = asaasID

	// 4. Processar Pagamento no Asaas
	var pixData *asaas.PixOutput
	var status string = "WAITING_PAYMENT"
	var asaasSubID string // <--- Precisamos capturar o ID da assinatura do Asaas

	if input.PaymentMethod == "PIX" {
		asaasSubID, pixData, err = uc.Gateway.SubscribePix(asaas.SubscribePixInput{
			CustomerID: asaasID,
			Price:      int64(plan.PriceCents),
		})
	} else {
		asaasSubID, _, err = uc.Gateway.Subscribe(asaas.SubscribeInput{
			CustomerID:       asaasID,
			Price:            float64(plan.PriceCents) / 100.0,
			CardNumber:       input.CardNumber,
			CardHolderName:   input.CardHolder,
			CardMonth:        input.CardMonth,
			CardYear:         input.CardYear,
			CardCCV:          input.CardCVV,
			HolderEmail:      input.Email,
			HolderCpfCnpj:    input.CPF,
			HolderPostalCode: input.ZipCode,
			HolderAddressNum: input.Number,
			HolderPhone:      input.Phone,
		})
	}

	if err != nil {
		// Se deu erro no pagamento, paramos aqui. Não salvamos nada no banco.
		return nil, fmt.Errorf("Asaas recusou o pagamento (400): %w", err)
	}

	// Atualizamos o cliente com o ID da assinatura externa (útil para auditoria)
	customer.SubscriptionID = asaasSubID

	// 5. SUCESSO NO ASAAS! Agora sim, salvamos tudo no Banco. 💾

	// A) Salva Cliente
	if err := uc.Repo.Create(ctx, customer); err != nil {
		// Logar erro crítico aqui (sistema ficou inconsistente: Asaas OK, Banco Fail)
		return nil, fmt.Errorf("erro crítico ao salvar cliente no banco: %w", err)
	}

	// B) Salva Assinatura (Interna)
	subscription := &entity.Subscription{
		ID:              uuid.New().String(),
		CustomerID:      customer.ID,
		PlanID:          plan.ID,
		ProductID:       plan.ProductID, // Não esqueça desse cara
		Amount:          plan.PriceCents,
		Status:          "PENDING",
		NextBillingDate: time.Now().AddDate(0, 1, 0),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := uc.SubRepo.Create(ctx, subscription); err != nil {
		return nil, fmt.Errorf("erro crítico ao salvar assinatura no banco: %w", err)
	}

	// 6. Retorna Sucesso
	var pixCode, pixUrl string
	if pixData != nil {
		pixCode = pixData.CopyPaste
		pixUrl = pixData.URL
	}

	return &CreateCustomerOutput{
		ID:           customer.ID,
		Status:       status,
		PixCode:      pixCode,
		PixQRCodeURL: pixUrl,
		Msg:          "Pré-cadastro realizado com sucesso!",
	}, nil
}

// Helperzinho para não quebrar se a data vier vazia
func parseDateOrNow(dateStr string) time.Time {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return time.Now()
	}
	return t
}
