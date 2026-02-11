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
	kommoService KommoService,
	welcomeBucketURL string,
	dependentRepo entity.DependentRepositoryInterface,
) *CreateCustomerUseCase {
	return &CreateCustomerUseCase{
		Repo:             repo,
		SubRepo:          subRepo,
		PlanRepo:         planRepo,
		Gateway:          gateway,
		Queue:            queue,
		EmailService:     emailService,
		KommoService:     kommoService,
		WelcomeBucketURL: welcomeBucketURL,
		DependentRepo:    dependentRepo,
	}
}

func (uc *CreateCustomerUseCase) Execute(ctx context.Context, input CreateCustomerInput) (*CreateCustomerOutput, error) {

	validationErrors := ValidateCreateCustomerInput(input)
	if len(validationErrors) > 0 {

		errMsg := "validation failed: "
		for _, e := range validationErrors {
			errMsg += e.Field + " (" + e.Message + "), "
		}
		return nil, &DomainError{
			Code:    "VALIDATION_ERROR",
			Message: errMsg,
		}
	}

	plan, err := uc.PlanRepo.FindByID(ctx, input.PlanID)
	if err != nil {
		return nil, &DomainError{
			Code:    "PLAN_NOT_FOUND",
			Message: "plano inválido: " + err.Error(),
		}
	}

	customerID := uuid.New().String()

	genderInt, _ := strconv.Atoi(input.Gender)
	if genderInt <= 0 || genderInt > 3 {
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

	customer := &entity.Customer{
		ID:        customerID,
		Name:      input.Name,
		Email:     input.Email,
		CPF:       input.CPF,
		Phone:     input.Phone,
		PlanID:    input.PlanID,
		ProductID: plan.ProductID,
		BirthDate: input.BirthDate,
		Gender:    genderInt,
		Address:   address,

		TermsAccepted:   input.TermsAccepted,
		TermsAcceptedAt: parseDateOrNow(input.TermsAcceptedAt),
		TermsVersion:    input.TermsVersion,

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

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

		return nil, &DomainError{
			Code:    "PAYMENT_FAILED",
			Message: "Asaas recusou o pagamento: " + err.Error(),
		}
	}

	customer.SubscriptionID = asaasSubID

	subscription := &entity.Subscription{
		ID:              uuid.New().String(),
		CustomerID:      customer.ID,
		PlanID:          plan.ID,
		ProductID:       plan.ProductID, // Não esqueça desse cara
		Amount:          plan.PriceCents,
		Status:          "PENDING",
		PaymentMethod:   input.PaymentMethod, // PIX, CREDIT_CARD
		NextBillingDate: time.Now().AddDate(0, 1, 0),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	txn := NewTransaction()

	txn.AddOperation("create_customer", func(ctx context.Context) error {
		return uc.Repo.Create(ctx, customer)
	})

	txn.AddCompensation("delete_customer", func(ctx context.Context) error {
		return uc.Repo.Delete(ctx, customer.ID)
	})

	txn.AddOperation("create_subscription", func(ctx context.Context) error {
		return uc.SubRepo.Create(ctx, subscription)
	})

	// Salvar dependentes (se houver)
	if len(input.Dependents) > 0 {
		txn.AddOperation("create_dependents", func(ctx context.Context) error {
			for _, depInput := range input.Dependents {
				genderInt, err := strconv.Atoi(depInput.Gender)
				if err != nil || genderInt < 1 || genderInt > 3 {
					genderInt = 1 // Default
				}

				dependent, err := entity.NewDependent(
					customer.ID,
					depInput.Name,
					depInput.CPF,
					depInput.BirthDate,
					genderInt,
					depInput.Kinship,
				)
				if err != nil {
					return fmt.Errorf("erro ao criar dependente %s: %w", depInput.Name, err)
				}

				if err := uc.DependentRepo.Create(ctx, dependent); err != nil {
					return fmt.Errorf("erro ao salvar dependente %s: %w", depInput.Name, err)
				}
			}
			return nil
		})
	}

	if err := txn.Execute(ctx); err != nil {
		return nil, &TechnicalError{
			Code:    "DATABASE_ERROR",
			Message: "failed to persist customer and subscription: " + err.Error(),
		}
	}

	// Notificações movidas para activate_subscription (após pagamento confirmado)

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

func parseDateOrNow(dateStr string) time.Time {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return time.Now()
	}
	return t
}
