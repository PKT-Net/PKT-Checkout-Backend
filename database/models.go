package database

import "time"

type Account struct {
	Id         uint32 `json:"id"`
	Merchant   string `json:"merchant"`
	ApiKey     string `json:"apiKey"`
	ViewKey    string `json:"viewKey"`
	SecretKey  string `json:"secretKey"`
	ColdWallet string `json:"coldWallet"`
}

type InvoiceStatus string

const (
	InvoiceStatusCreated InvoiceStatus = "created"
	InvoiceStatusPending InvoiceStatus = "pending"
	InvoiceStatusExpired InvoiceStatus = "expired"
	InvoiceStatusPaid    InvoiceStatus = "paid"
)

type Invoice struct {
	Id                 string        `json:"id"`
	ClientId           string        `json:"clientId"`
	AccountId          uint32        `json:"accountId"`
	PaymentAmount      uint64        `json:"paymentAmount"`
	PaymentAddress     string        `json:"paymentAddress"`
	PaymentDescription string        `json:"paymentDescription"`
	CallbackUrl        string        `json:"callbackUrl"`
	CreationTime       time.Time     `json:"creationTime"`
	ExpirationTime     time.Time     `json:"expirationTime"`
	Status             InvoiceStatus `json:"status"`
}

type WalletTransaction struct {
	Id               string    `json:"id"`
	InvoiceId        string    `json:"invoiceId"`
	WalletAddress    string    `json:"walletAddress"`
	PaymentAmount    uint64    `json:"paymentAmount"`
	ConfirmationTime time.Time `json:"confirmationTime"`
	DiscoveryTime    time.Time `json:"discoveryTime"`
}

type CallbackStatus string

const (
	CallbackStatusCreated   CallbackStatus = "created"
	CallbackStatusFailed    CallbackStatus = "failed"
	CallbackStatusDelivered CallbackStatus = "delivered"
)

type Callback struct {
	Id          string         `json:"id"`
	InvoiceId   string         `json:"invoiceId"`
	RequestTime time.Time      `json:"requestTime"`
	NextReqTime time.Time      `json:"nextReqTime"`
	ReqErrors   int            `json:"reqErrors"`
	Status      CallbackStatus `json:"status"`
}

func (i *Invoice) Save() error {
	dbConnection := GetConnection()

	_, err := dbConnection.Exec("INSERT INTO invoices (id, clientId, accountId, paymentAmount, paymentAddress, paymentDescription, callbackUrl, creationTime, expirationTime, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", i.Id, i.ClientId, i.AccountId, i.PaymentAmount, i.PaymentAddress, i.PaymentDescription, i.CallbackUrl, i.CreationTime, i.ExpirationTime, i.Status)
	if err != nil {
		return err
	}

	return nil
}

func (i *Invoice) Update() error {
	dbConnection := GetConnection()

	_, err := dbConnection.Exec("UPDATE invoices SET status = ?, callbackUrl = ? WHERE id = ? ", i.Status, i.CallbackUrl, i.Id)
	if err != nil {
		return err
	}

	return nil
}

func (w *WalletTransaction) Save() error {
	dbConnection := GetConnection()

	_, err := dbConnection.Exec("INSERT INTO walletTransactions (id, invoiceId, walletAddress, paymentAmount, confirmationTime, discoveryTime) VALUES (?, ?, ?, ?, ?, ?)", w.Id, w.InvoiceId, w.WalletAddress, w.PaymentAmount, w.ConfirmationTime, w.DiscoveryTime)
	if err != nil {
		return err
	}

	return nil
}

func (c *Callback) Save() error {
	dbConnection := GetConnection()

	_, err := dbConnection.Exec("INSERT INTO callbacks (id, invoiceId, requestTime, nextReqTime, reqErrors, status) VALUES (?, ?, ?, ?, ?, ?)", c.Id, c.InvoiceId, c.RequestTime, c.NextReqTime, c.ReqErrors, c.Status)
	if err != nil {
		return err
	}

	return nil
}

func (c *Callback) Update() error {
	dbConnection := GetConnection()

	_, err := dbConnection.Exec("UPDATE callbacks SET nextReqTime = ?, reqErrors = ?, status = ? WHERE id = ? ", c.NextReqTime, c.ReqErrors, c.Status, c.Id)
	if err != nil {
		return err
	}

	return nil
}
