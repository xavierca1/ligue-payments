package temsaude

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/xavierca1/ligue-payments/internal/entity"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

















































































func (c *Client) RegisterBeneficiary(ctx context.Context, customer *entity.Customer) (string, error) {

	fmt.Println("\n===============================================================")
	fmt.Printf("🚧 MOCK ALERT: Simulating Tem Saude registration for [%s]\n", customer.Name)
	fmt.Println("   Reason: Legacy endpoint is returning 404/DNS Error.")
	fmt.Println("   Returning fake ProviderID: 'MOCK-TEM-123456'")
	fmt.Println("===============================================================\n")


	return "MOCK-TEM-123456", nil
}
