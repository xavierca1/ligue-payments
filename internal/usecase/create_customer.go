package usecase

import (
	"context"
	"fmt"
	"strconv"

	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/asaas"
	"github.com/xavierca1/ligue-payments/internal/infra/queue"
)

func NewCreateCustomerUseCase(
	repo CustomerRepositoryInterface,
	planRepo PlanRepositoryInterface,
	gateway PaymentGateway,
	queue QueueProducerInterface,
	// benefitService BenefitProvider,
	emailService EmailService,
	welcomeBucketURL string,
) *CreateCustomerUseCase {
	return &CreateCustomerUseCase{
		Repo:     repo,
		PlanRepo: planRepo,
		Gateway:  gateway,
		// BenefitService: benefitService,
		Queue:            queue,
		EmailService:     emailService,
		WelcomeBucketURL: welcomeBucketURL,
	}
}

func (uc *CreateCustomerUseCase) Execute(ctx context.Context, input CreateCustomerInput) (*CreateCustomerOutput, error) {
	genderInt, _ := strconv.Atoi(input.Gender)

	plan, err := uc.PlanRepo.FindByID(ctx, input.PlanID)
	if err != nil {
		return nil, fmt.Errorf("plano inválido ou não encontrado: %w", err)
	}

	asaasCustomerID, err := uc.Gateway.CreateCustomer(asaas.CreateCustomerInput{
		Name:          input.Name,
		Email:         input.Email,
		CpfCnpj:       input.CPF,
		Phone:         input.Phone,
		MobilePhone:   input.Phone,
		PostalCode:    input.ZipCode,
		AddressNumber: input.Number,
	})
	if err != nil {
		return nil, fmt.Errorf("erro ao criar cliente no asaas: %w", err)
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

	customer, err := entity.NewCustomer(
		input.Name, input.Email, input.CPF, input.OnixCode,
		input.Phone, input.BirthDate, genderInt, address,
	)
	if err != nil {
		return nil, err
	}

	customer.GatewayID = asaasCustomerID

	amount := float64(plan.PriceCents) / 100.0

	var pixCode, pixQRCode string

	if input.PaymentMethod == "PIX" {

		subID, pixData, err := uc.Gateway.SubscribePix(asaas.SubscribePixInput{
			CustomerID: asaasCustomerID,
			Price:      int64(plan.PriceCents),
		})
		if err != nil {
			return nil, fmt.Errorf("erro ao gerar PIX: %w", err)
		}

		customer.SubscriptionID = subID
		customer.Status = "WAITING_PAYMENT"

		pixCode = pixData.CopyPaste
		pixQRCode = pixData.URL

	} else {

		subID, status, err := uc.Gateway.Subscribe(asaas.SubscribeInput{
			CustomerID:       asaasCustomerID,
			Price:            amount,
			CardHolderName:   input.CardHolder,
			CardNumber:       input.CardNumber,
			CardMonth:        input.CardMonth,
			CardYear:         input.CardYear,
			CardCCV:          input.CardCVV,
			HolderEmail:      input.Email,
			HolderCpfCnpj:    input.CPF,
			HolderPostalCode: input.ZipCode,
			HolderAddressNum: input.Number,
			HolderPhone:      input.Phone,
		})

		if err != nil {
			return nil, fmt.Errorf("erro no pagamento via cartão: %w", err)
		}

		customer.SubscriptionID = subID
		customer.Status = status // Ex: "ACTIVE" ou "CONFIRMED"

		// B. MANDA PARA A FILA (RABBITMQ) 🐰
		// O pagamento passou, então avisamos o Worker para ativar na Tem/Doc24
		payload := queue.ActivationPayload{
			CustomerID: customer.ID,
			PlanID:     plan.ID,
			Provider:   plan.Provider, // "TEM_SAUDE" ou "DOC24"
			Name:       customer.Name,
			Email:      customer.Email,
			Origin:     "CHECKOUT_CREDIT_CARD",
		}

		err = uc.Queue.PublishActivation(ctx, payload)
		if err != nil {
			// Logamos o erro, mas NÃO travamos a venda. O dinheiro já foi capturado.
			fmt.Printf("⚠️ ERRO CRÍTICO: Falha ao publicar na fila RabbitMQ: %v\n", err)
			// Opcional: Poderia salvar em uma tabela de 'retry_queue' no banco
		}
	}

	err = uc.Repo.Create(ctx, customer)
	if err != nil {
		return nil, fmt.Errorf("erro ao salvar cliente no banco: %w", err)
	}

	go func() {
		if customer.Status != "WAITING_PAYMENT" {

			pdfLink := fmt.Sprintf("%s/kit_%s.pdf", uc.WelcomeBucketURL, plan.ProviderPlanCode)

			err := uc.EmailService.SendWelcome(input.Email, input.Name, plan.Name, pdfLink)

			if err != nil {
				fmt.Printf("⚠️ FALHA EMAIL: %v\n", err)
			} else {
				fmt.Printf("📧 Email enviado para %s\n", input.Email)
			}
		}
	}()

	return &CreateCustomerOutput{
		ID:           customer.ID,
		Name:         customer.Name,
		Email:        customer.Email,
		Status:       customer.Status,
		Msg:          "Operação realizada com sucesso",
		PixCode:      pixCode,   // Cheio se for PIX, Vazio se for Cartão
		PixQRCodeURL: pixQRCode, // Cheio se for PIX, Vazio se for Cartão
	}, nil
}
