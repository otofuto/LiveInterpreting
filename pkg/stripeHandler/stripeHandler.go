package stripeHandler

import (
	"errors"
	"fmt"
	"log"
	"os"

	stripe "github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/paymentintent"
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

func Payment(cusid string, yen int64) error {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	cus, err := customer.Get(cusid, nil)
	if err != nil {
		log.Println("stripeHandler.go Payment(cusid string, yen int)")
		log.Println(err)
		return err
	}
	a := cus.InvoiceSettings.DefaultPaymentMethod
	fmt.Println(a.ID)
	pms := &stripe.PaymentIntentParams{
		Amount:             stripe.Int64(yen),
		Currency:           stripe.String(string(stripe.CurrencyJPY)),
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		Customer:           &cusid,
		PaymentMethod:      stripe.String(a.ID),
	}
	pi, err := paymentintent.New(pms)
	if err != nil {
		log.Println("stripeHandler.go Payment(cusid string, yen int)")
		log.Println(err)
		return err
	}
	fmt.Println(pi.ID)
	fmt.Println(pi.Status) //requires_confirmation
	return errors.New("undefined error")
}

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

	isp := &stripe.CustomerInvoiceSettingsParams{
		DefaultPaymentMethod: &token,
	}
	params := &stripe.CustomerParams{
		Email:           stripe.String(email),
		Name:            stripe.String(name),
		PaymentMethod:   &token,
		InvoiceSettings: isp,
		//Source: &stripe.SourceParams{
		//	Token: stripe.String(token),
		//},
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
