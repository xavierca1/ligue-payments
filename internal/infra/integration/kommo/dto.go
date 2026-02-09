package kommo

type CreateLeadInput struct {
	CustomerName string
	Phone        string
	Email        string
	PlanName     string
	Price        int
	Origin       string // Canal de origem (ex: "Checkout Website")
}
