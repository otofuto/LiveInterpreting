package stripeHandler

import (
	"fmt"
	"log"
	"os"

	stripe "github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/setupintent"
)

type Webhook struct {
	Type string      `json:"type"`
	Data WebhookData `json:"data"`
}

type WebhookData struct {
	Object WebhookObject `json:"object"`
}

type WebhookObject struct {
	Id       string `json:"id"`
	Object   string `json:"object"`
	Customer string `json:"customer"`
}

// func Test() (*stripe.PaymentIntent, error) {
// 	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
// 	params := &stripe.PaymentIntentParams {
// 		Amount: stripe.Int64(15000),
// 		Currency: stripe.String(string(stripe.CurrencyJPY)),
// 		PaymentMethodTypes: stripe.StringSlice([]string {"card"}),
// 	}
// 	return paymentintent.New(params)
// }

/*func Payment(cusid string, yen int) error {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	params := &stripe.PaymentIntentParams{
		Amount:             stripe.Int64(int64(yen)),
		Currency:           stripe.String(string(stripe.CurrencyJPY)),
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		Customer:           &cusid,
	}
	pi, err := paymentintent.New(params)
	if err != nil {
		log.Println(err)
		return err
	}
	fmt.Println(pi.ID)
	fmt.Println(pi.ClientSecret)
	return nil
}*/

func GetClientSecret() string {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	pms := &stripe.SetupIntentParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
	}
	si, err := setupintent.New(pms)
	if err != nil {
		log.Println(err)
		return ""
	}
	return si.ClientSecret
}

func CreateCustomer(email, name, token string) string {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
		Source: &stripe.SourceParams{
			Token: stripe.String(token),
		},
	}
	cus, err := customer.New(params)
	if err != nil {
		log.Println(err)
		return ""
	}
	fmt.Println(cus.ID)
	return cus.ID
}

func DeleteCustomer(cusId string) error {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	_, err := customer.Del(cusId, nil)
	return err
}