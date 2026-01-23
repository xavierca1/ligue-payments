package usecase

import (
	"context"
	"fmt"
	"strconv"

	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/asaas"
)

func NewCreateCustomerUseCase(
	repo CustomerRepositoryInterface,
	subRepo SubscriptionRepository, // 👈 ADICIONADO: Argumento novo
	planRepo PlanRepositoryInterface,
	gateway PaymentGateway,
	queue QueueProducerInterface,
	emailService EmailService,
	welcomeBucketURL string,
) *CreateCustomerUseCase {
	return &CreateCustomerUseCase{
		Repo:             repo,
		SubRepo:          subRepo, // 👈 ADICIONADO: Injeção
		PlanRepo:         planRepo,
		Gateway:          gateway,
		Queue:            queue,
		EmailService:     emailService,
		WelcomeBucketURL: welcomeBucketURL,
	}
}

func (uc *CreateCustomerUseCase) Execute(ctx context.Context, input CreateCustomerInput) (*CreateCustomerOutput, error) {
	plan, err := uc.PlanRepo.FindByID(ctx, input.PlanID)
	if err != nil {
		return nil, fmt.Errorf("plano inválido: %w", err)
	}

	asaasID, err := uc.Gateway.CreateCustomer(asaas.CreateCustomerInput{
		Name: input.Name, Email: input.Email, CpfCnpj: input.CPF,
		Phone: input.Phone, PostalCode: input.ZipCode,
	})
	if err != nil {
		return nil, err
	}

	genderInt, _ := strconv.Atoi(input.Gender)
	address := entity.Address{
		Street: input.Street, Number: input.Number, Complement: input.Complement,
		District: input.District, City: input.City, State: input.State, ZipCode: input.ZipCode,
	}

	customer, err := entity.NewCustomer(
		input.Name, input.Email, input.CPF, input.OnixCode, // Ajuste se não tiver OnixCode na entity
		input.Phone, input.BirthDate, genderInt, address,
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar entidade cliente: %w", err)
	}

	customer.GatewayID = asaasID

	if err := uc.Repo.Create(ctx, customer); err != nil {
		return nil, fmt.Errorf("falha ao salvar cliente: %w", err)
	}

	var pixData *asaas.PixOutput
	var status string = "WAITING_PAYMENT"

	if input.PaymentMethod == "PIX" {
		_, pixData, err = uc.Gateway.SubscribePix(asaas.SubscribePixInput{
			CustomerID: asaasID,
			Price:      int64(plan.PriceCents),
		})
	} else {
		_, _, err = uc.Gateway.Subscribe(asaas.SubscribeInput{
			CustomerID: asaasID,
			Price:      float64(plan.PriceCents) / 100,

			CardNumber:     input.CardNumber,
			CardHolderName: input.CardHolder, // Nome impresso no cartão
			CardMonth:      input.CardMonth,  // Ex: "12"
			CardYear:       input.CardYear,   // Ex: "2030"
			CardCCV:        input.CardCVV,    // Código de segurança

			HolderEmail:      input.Email,
			HolderCpfCnpj:    input.CPF,
			HolderPostalCode: input.ZipCode,
			HolderAddressNum: input.Number,
			HolderPhone:      input.Phone,
		})
	}

	if err != nil {
		fmt.Printf(" Falha no pagamento. Iniciando Rollback do cliente %s...\n", customer.ID)

		rollbackErr := uc.Repo.Delete(ctx, customer.ID)
		if rollbackErr != nil {

			fmt.Printf("CRITICAL: Falha ao fazer rollback do cliente: %v\n", rollbackErr)
		} else {
			fmt.Printf(" Rollback concluído. Cliente removido.\n")
		}

		return nil, fmt.Errorf("erro ao processar pagamento no gateway: %w", err)
	}

	subscription := entity.NewSubscription(customer.ID, plan.ID, plan.PriceCents)

	if err := uc.SubRepo.Create(ctx, subscription); err != nil {
		return nil, fmt.Errorf("erro ao criar assinatura: %w", err)
	}

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
		Msg:          "Pré-cadastro realizado. Aguardando pagamento.",
	}, nil
}
