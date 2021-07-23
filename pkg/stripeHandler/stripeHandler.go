package stripeHandler

import (
	"os"
	"fmt"
	"log"
	stripe "github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/sub"
	paymentintent "github.com/stripe/stripe-go/v72/paymentintent"
)

type Webhook struct {
	Type string `json:"type"`
	Data WebhookData `json:"data"`
}

type WebhookData struct {
	Object WebhookObject `json:"object"`
}

type WebhookObject struct {
	Id string `json:"id"`
	Object string `json:"object"`
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

func GetClientSecret(yen int) string {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	params := &stripe.PaymentIntentParams {
		Amount: stripe.Int64(int64(yen)),
		Currency: stripe.String(string(stripe.CurrencyJPY)),
		PaymentMethodTypes: stripe.StringSlice([]string {"card"}),
	}
	pi, err := paymentintent.New(params)
	if err != nil {
		log.Println(err)
		return ""
	}
	fmt.Println(pi.ID)
	return pi.ClientSecret
	return ""
}

func CreateCustomer(email, name, token string) string {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	params := &stripe.CustomerParams {
		Email: stripe.String(email),
		Name: stripe.String(name),
		Source: &stripe.SourceParams {
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

func GetSubscription(cus, priceId string) string {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	params := &stripe.SubscriptionParams {
		Customer: stripe.String(cus),
		Items: []*stripe.SubscriptionItemsParams {
			{
				Price: stripe.String(priceId),
			},
		},
	}
	s, err := sub.New(params)
	if err != nil {
		log.Println(err)
		return ""
	}
	fmt.Println(s.ID)
	return s.ID
}

func EndSubscription(sid string) error {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	params := &stripe.SubscriptionParams {
		CancelAtPeriodEnd: stripe.Bool(true),
	}
	params.AddMetadata("cancel_at_period_end", "true")
	_, err := sub.Update(sid, params)
	if err != nil {
		return err
	}
	return nil
}
