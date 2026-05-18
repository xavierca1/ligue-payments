package usecase

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xavierca1/ligue-payments/internal/entity"
	"github.com/xavierca1/ligue-payments/internal/infra/integration/asaas"
)

func parseDependentGender(value string) (int, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "1", "M", "MASCULINO":
		return 1, nil
	case "2", "F", "FEMININO":
		return 2, nil
	case "3", "O", "OUTRO", "OTHER":
		return 3, nil
	default:
		return 0, fmt.Errorf("gênero do dependente inválido: %s", value)
	}
}

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

	// 1. Validação de Input
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

	// 2. Busca e Validação do Plano
	plan, err := uc.PlanRepo.FindByID(ctx, input.PlanID)
	if err != nil {
		return nil, &DomainError{
			Code:    "PLAN_NOT_FOUND",
			Message: "plano inválido: " + err.Error(),
		}
	}

	// 3. Lógica de Valores e Cupons
	originalAmountCents := plan.PriceCents
	finalAmountCents := originalAmountCents
	discountPercent := 0
	discountAmountCents := 0
	couponCode := strings.ToUpper(strings.TrimSpace(input.CouponCode))
	couponSellerName := ""

	if couponCode != "" {
		if couponTracker == nil {
			return nil, &DomainError{
				Code:    "COUPON_UNAVAILABLE",
				Message: "cupom informado, mas o serviço de cupom não está configurado",
			}
		}
		couponDetails, couponErr := couponTracker.GetActiveCoupon(ctx, couponCode)
		if couponErr != nil {
			return nil, &DomainError{
				Code:    "COUPON_INVALID",
				Message: "cupom inválido ou inativo",
			}
		}

		discountPercent = couponDetails.DiscountPercent
		if discountPercent <= 0 {
			discountPercent = 10
		}
		couponSellerName = couponDetails.SellerName

		discountAmountCents = (originalAmountCents * discountPercent) / 100
		finalAmountCents = originalAmountCents - discountAmountCents
		if finalAmountCents < 0 {
			finalAmountCents = 0
		}
	}

	// 4. Busca de Cliente e Assinatura Existente
	existingCustomer, _ := uc.Repo.FindByCPF(ctx, input.CPF)
	if existingCustomer == nil {
		existingCustomer, _ = uc.Repo.FindByEmailAndProductID(ctx, input.Email, plan.ProductID)
	}

	var latestSubscription *entity.Subscription
	if existingCustomer != nil {
		latestSubscription, _ = uc.SubRepo.FindLastByCustomerID(ctx, existingCustomer.ID)
	}

	// 5. Máquina de Estados: Resolução de Status (ACTIVE vs PENDING)
	if existingCustomer != nil {
		existingStatus := strings.ToUpper(strings.TrimSpace(existingCustomer.Status))

		if latestSubscription != nil {
			existingStatus = strings.ToUpper(strings.TrimSpace(latestSubscription.Status))
		}

		if existingStatus == "" {
			if subStatus, subErr := uc.SubRepo.GetStatusByCustomerID(existingCustomer.ID); subErr == nil {
				existingStatus = strings.ToUpper(strings.TrimSpace(subStatus))
			}
		}

		if existingStatus == "" {
			existingStatus = "PENDING"
		}

		if existingStatus == "ACTIVE" {
			cpfClean := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(input.CPF, ".", ""), "-", ""), " ", "")
			cpfMasked := "***"
			if len(cpfClean) >= 4 {
				cpfMasked = strings.Repeat("*", len(cpfClean)-4) + cpfClean[len(cpfClean)-4:]
			}
			log.Printf("[checkout] blocked: customer already active cpf=%s email=%s", cpfMasked, input.Email)
			return nil, &DomainError{
				Code:    "CUSTOMER_ALREADY_ACTIVE",
				Message: "Já existe uma assinatura ativa para este CPF. Em caso de dúvidas, entre em contato com o SAC.",
			}
		}

		if existingStatus == "PENDING" {
			// Cancela a assinatura anterior no Asaas antes de reprocessar
			subIDToCancel := ""
			if latestSubscription != nil && strings.TrimSpace(latestSubscription.PaymentMethodID) != "" {
				subIDToCancel = latestSubscription.PaymentMethodID
			} else if strings.TrimSpace(existingCustomer.SubscriptionID) != "" {
				subIDToCancel = existingCustomer.SubscriptionID
			}

			if subIDToCancel != "" {
				if cancelErr := uc.Gateway.DeleteSubscription(subIDToCancel); cancelErr != nil {
					log.Printf("[WARN] falha ao cancelar assinatura Asaas %s: %v", subIDToCancel, cancelErr)
				}
			}

			if latestSubscription != nil {
				if deleteErr := uc.SubRepo.DeleteByID(ctx, latestSubscription.ID); deleteErr != nil {
					log.Printf("[WARN] falha ao deletar subscription %s: %v", latestSubscription.ID, deleteErr)
				}
			}
			if deleteErr := uc.Repo.Delete(ctx, existingCustomer.ID); deleteErr != nil {
				log.Printf("[WARN] falha ao deletar customer %s: %v", existingCustomer.ID, deleteErr)
			}

			existingCustomer = nil
			latestSubscription = nil
		}
	}

	// 6. CRIAÇÃO DO NOVO CLIENTE (Se não existir)
	newCustomerCreated := false
	if existingCustomer == nil {
		// Converter Gender de string para int
		genderInt, genderErr := parseDependentGender(input.Gender)
		if genderErr != nil {
			return nil, &DomainError{Code: "VALIDATION_ERROR", Message: genderErr.Error()}
		}

		existingCustomer = &entity.Customer{
			ID:            uuid.New().String(),
			Name:          input.Name,
			Email:         input.Email,
			CPF:           input.CPF,
			Phone:         input.Phone,
			BirthDate:     input.BirthDate,
			Gender:        genderInt,
			MaritalStatus: input.MaritalStatus,
			Status:        "PENDING",
			ProductID: plan.ProductID,
			PlanID:    plan.ID,
			Address: entity.Address{
				Street:     input.Street,
				Number:     input.Number,
				Complement: input.Complement,
				District:   input.District,
				City:       input.City,
				State:      input.State,
				ZipCode:    input.ZipCode,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		newCustomerCreated = true

		log.Printf("[DEBUG] ID do cliente sendo enviado para o banco: '%s'", existingCustomer.ID)
		if err := uc.Repo.Create(ctx, existingCustomer); err != nil {
			log.Printf("[ERROR] Failed to create customer: %v", err)
			return nil, &TechnicalError{Code: "DATABASE_ERROR", Message: "Falha ao criar cliente"}
		}
	}

	if uc.DependentRepo != nil && len(input.Dependents) > 0 {
		for _, dependentInput := range input.Dependents {
			gender, genderErr := parseDependentGender(dependentInput.Gender)
			if genderErr != nil {
				if newCustomerCreated {
					if deleteErr := uc.Repo.Delete(ctx, existingCustomer.ID); deleteErr != nil {
						log.Printf("[WARN] rollback delete customer failed after dependent validation: %v", deleteErr)
					}
				}
				return nil, &DomainError{Code: "VALIDATION_ERROR", Message: genderErr.Error()}
			}

			dependent, dependentErr := entity.NewDependent(existingCustomer.ID, existingCustomer.CPF, dependentInput.Name, dependentInput.CPF, dependentInput.BirthDate, gender, dependentInput.Kinship)
			if dependentErr != nil {
				if newCustomerCreated {
					if deleteErr := uc.Repo.Delete(ctx, existingCustomer.ID); deleteErr != nil {
						log.Printf("[WARN] rollback delete customer failed after dependent creation error: %v", deleteErr)
					}
				}
				return nil, &DomainError{Code: "VALIDATION_ERROR", Message: dependentErr.Error()}
			}

			if err := uc.DependentRepo.Create(ctx, dependent); err != nil {
				if newCustomerCreated {
					if deleteErr := uc.Repo.Delete(ctx, existingCustomer.ID); deleteErr != nil {
						log.Printf("[WARN] rollback delete customer failed after dependent persist error: %v", deleteErr)
					}
				}
				return nil, &TechnicalError{Code: "DATABASE_ERROR", Message: fmt.Sprintf("Falha ao criar dependente %s", dependent.Name)}
			}
		}
	}

	asaasCustomerID := strings.TrimSpace(existingCustomer.GatewayID)
	if asaasCustomerID == "" {
		createdGatewayID, gatewayErr := uc.Gateway.CreateCustomer(asaas.CreateCustomerInput{
			Name:              input.Name,
			Email:             input.Email,
			CpfCnpj:           input.CPF,
			Phone:             input.Phone,
			PostalCode:        input.ZipCode,
			AddressNumber:     input.Number,
			ExternalReference: existingCustomer.ID,
		})
		if gatewayErr != nil {
			return nil, &TechnicalError{Code: "GATEWAY_ERROR", Message: "Falha ao criar customer no gateway: " + gatewayErr.Error()}
		}
		asaasCustomerID = createdGatewayID
		existingCustomer.GatewayID = asaasCustomerID
		// Salvar o gateway_id no banco para que o webhook consiga encontrar o cliente depois
		if updateErr := uc.Repo.UpdateGatewayID(ctx, existingCustomer.ID, asaasCustomerID); updateErr != nil {
			log.Printf("[WARN] Falha ao atualizar gateway_id no banco: %v", updateErr)
			// Não bloqueamos - o webhook pode tentar encontrar de outras formas
		}
	}

	paymentMethod := strings.ToUpper(strings.TrimSpace(input.PaymentMethod))
	var gatewaySubscriptionID string
	var pixData *asaas.PixOutput
	var gatewayStatus string
	var gatewayErr error

	if paymentMethod == "PIX" {
		gatewaySubscriptionID, pixData, gatewayErr = uc.Gateway.SubscribePix(asaas.SubscribePixInput{
			CustomerID: asaasCustomerID,
			Price:      int64(finalAmountCents),
		})
		gatewayStatus = "PENDING"
	} else if paymentMethod == "CREDIT_CARD" {
		gatewaySubscriptionID, gatewayStatus, gatewayErr = uc.Gateway.Subscribe(asaas.SubscribeInput{
			CustomerID:       asaasCustomerID,
			Price:            float64(finalAmountCents) / 100.0,
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
	} else {
		return nil, &DomainError{Code: "UNSUPPORTED_PAYMENT", Message: "Método não suportado"}
	}

	if gatewayErr != nil {
		return nil, &DomainError{Code: "PAYMENT_FAILED", Message: "Asaas recusou o pagamento: " + gatewayErr.Error()}
	}

	existingCustomer.SubscriptionID = gatewaySubscriptionID
	if paymentMethod == "CREDIT_CARD" {
		existingCustomer.Status = strings.ToUpper(strings.TrimSpace(gatewayStatus))
	}

	newSubscription := &entity.Subscription{
		ID:              uuid.New().String(),
		CustomerID:      existingCustomer.ID,
		PlanID:          plan.ID,
		ProductID:       plan.ProductID,
		Amount:          finalAmountCents,
		Status:          gatewayStatus,
		PaymentMethod:   paymentMethod,
		PaymentMethodID: gatewaySubscriptionID,
		NextBillingDate: time.Now().AddDate(0, 1, 0),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := uc.SubRepo.Create(ctx, newSubscription); err != nil {
		log.Printf("[ERROR] Failed to create subscription: %v", err)
		if newCustomerCreated {
			if deleteErr := uc.Repo.Delete(ctx, existingCustomer.ID); deleteErr != nil {
				log.Printf("[WARN] rollback delete customer failed: %v", deleteErr)
			}
		}
		return nil, &TechnicalError{Code: "DATABASE_ERROR", Message: "Falha ao criar assinatura"}
	}

	if couponCode != "" && couponTracker != nil {
		if trackErr := couponTracker.TrackSale(ctx, CouponSaleRecord{
			CouponCode:          couponCode,
			SellerName:          couponSellerName,
			CustomerID:          existingCustomer.ID,
			SubscriptionID:      newSubscription.ID,
			PlanID:              plan.ID,
			OriginalAmountCents: originalAmountCents,
			DiscountPercent:     discountPercent,
			DiscountAmountCents: discountAmountCents,
			FinalAmountCents:    finalAmountCents,
		}); trackErr != nil {
			log.Printf("[WARN] falha ao registrar venda do cupom %s: %v", couponCode, trackErr)
		}
	}

	if paymentMethod == "PIX" {
		if pixData == nil {
			return nil, &DomainError{Code: "PAYMENT_FAILED", Message: "Asaas não retornou o QR Code do PIX"}
		}
		log.Printf("[checkout] pix_ready customer_id=%s sub_id=%s code_len=%d qr_len=%d", existingCustomer.ID, gatewaySubscriptionID, len(strings.TrimSpace(pixData.CopyPaste)), len(strings.TrimSpace(pixData.URL)))

		return &CreateCustomerOutput{
			ID:           existingCustomer.ID,
			Status:       "WAITING_PAYMENT",
			PixCode:      pixData.CopyPaste,
			PixQRCodeURL: pixData.URL,
			Msg:          "Cobrança gerada com sucesso!",
		}, nil
	}

	return &CreateCustomerOutput{
		ID:     existingCustomer.ID,
		Status: strings.ToUpper(strings.TrimSpace(gatewayStatus)),
		Msg:    "Pagamento processado com sucesso!",
	}, nil
}
