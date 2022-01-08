package stripeHandler

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/otofuto/LiveInterpreting/pkg/database/accounts"
	stripe "github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/account"
	"github.com/stripe/stripe-go/v72/accountlink"
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
		Confirm:            stripe.Bool(true),
	}
	pi, err := paymentintent.New(pms)
	if err != nil {
		log.Println("stripeHandler.go Payment(cusid string, yen int)")
		log.Println(err)
		return err
	}
	fmt.Println(pi.ID)
	fmt.Println(pi.Status)
	if pi.Status == stripe.PaymentIntentStatusSucceeded {
		return nil
	}
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

func CreateAccount(ac *accounts.Accounts) string {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	pms := &stripe.AccountParams{
		Country:      stripe.String("JP"),
		Email:        stripe.String(ac.Email),
		Type:         stripe.String("standard"),
		BusinessType: stripe.String("individual"),
		BusinessProfile: &stripe.AccountBusinessProfileParams{
			Name:               stripe.String(ac.Name),
			ProductDescription: stripe.String("We will be an interpreter for live streaming at the request of influencers."),
			SupportAddress: &stripe.AddressParams{
				City:       stripe.String("渋谷区"),
				Country:    stripe.String("jp"),
				Line1:      stripe.String("渋谷2-14-6"),
				Line2:      stripe.String("西田ビル5F"),
				PostalCode: stripe.String("150-0002"),
				State:      stripe.String("東京都"),
			},
			SupportEmail: stripe.String(ac.Email),
			SupportPhone: stripe.String("08061234039"),
			//SupportURL:         stripe.String(os.Getenv("HOST") + "/u/" + strconv.Itoa(ac.Id)),
			SupportURL: stripe.String("https://live-interpreting.herokuapp.com/u/" + strconv.Itoa(ac.Id)),
			//URL:                stripe.String(os.Getenv("HOST") + "/u/" + strconv.Itoa(ac.Id)),
			URL: stripe.String("https://live-interpreting.herokuapp.com/u/" + strconv.Itoa(ac.Id)),
		},
		Individual: &stripe.PersonParams{
			Address: &stripe.AccountAddressParams{
				Country: stripe.String("jp"),
			},
			Email:             stripe.String(ac.Email),
			PoliticalExposure: stripe.String("none"),
		},
		DefaultCurrency: stripe.String("JPY"),
	}
	a, err := account.New(pms)
	if err != nil {
		log.Println(err)
		return ""
	}
	return a.ID
}

func CreateAccountLink(aid string) string {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	params := &stripe.AccountLinkParams{
		Account:    stripe.String(aid),
		RefreshURL: stripe.String(os.Getenv("HOST") + "/connect/create/"),
		ReturnURL:  stripe.String(os.Getenv("HOST") + "/connect/success/"),
		Type:       stripe.String("account_onboarding"),
	}
	al, err := accountlink.New(params)
	if err != nil {
		log.Println(err)
		return ""
	}
	return al.URL
}

func GetAccount(aid string) (*stripe.Account, error) {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	ac, err := account.GetByID(aid, nil)
	if err != nil {
		return &stripe.Account{}, err
	}
	return ac, nil
}

func DeleteAccount(aid string) error {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	_, err := account.Del(aid, nil)
	if err != nil {
		return err
	}
	return nil
}
