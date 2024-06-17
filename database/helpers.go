package database

func FetchAccountById(id uint32) (Account, error) {
	var account Account
	dbConnection := GetConnection()
	if err := dbConnection.QueryRow("SELECT id, merchant, apiKey, viewKey, secretKey, coldWallet FROM accounts WHERE id = ?", id).Scan(&account.Id, &account.Merchant, &account.ApiKey, &account.ViewKey, &account.SecretKey, &account.ColdWallet); err != nil {
		return account, err
	}
	return account, nil
}

func FetchAccountByApiKey(apiKey string) (Account, error) {
	var account Account
	dbConnection := GetConnection()
	if err := dbConnection.QueryRow("SELECT id, merchant, apiKey, viewKey, secretKey, coldWallet FROM accounts WHERE apiKey = ?", apiKey).Scan(&account.Id, &account.Merchant, &account.ApiKey, &account.ViewKey, &account.SecretKey, &account.ColdWallet); err != nil {
		return account, err
	}
	return account, nil
}

func FetchAccountByViewKey(viewKey string) (Account, error) {
	var account Account
	dbConnection := GetConnection()
	if err := dbConnection.QueryRow("SELECT id, merchant, apiKey, viewKey, secretKey, coldWallet FROM accounts WHERE viewKey = ?", viewKey).Scan(&account.Id, &account.Merchant, &account.ApiKey, &account.ViewKey, &account.SecretKey, &account.ColdWallet); err != nil {
		return account, err
	}
	return account, nil
}

func FetchInvoiceById(id string) (Invoice, error) {
	var invoice Invoice
	dbConnection := GetConnection()
	if err := dbConnection.QueryRow("SELECT id, clientId, accountId, paymentAmount, paymentAddress, paymentDescription, callbackUrl, creationTime, expirationTime, status FROM invoices WHERE id = ?", id).Scan(&invoice.Id, &invoice.ClientId, &invoice.AccountId, &invoice.PaymentAmount, &invoice.PaymentAddress, &invoice.PaymentDescription, &invoice.CallbackUrl, &invoice.CreationTime, &invoice.ExpirationTime, &invoice.Status); err != nil {
		return invoice, err
	}
	return invoice, nil
}

func FetchPendingInvoices() ([]Invoice, error) {
	var invoices []Invoice
	dbConnection := GetConnection()
	rows, err := dbConnection.Query("SELECT id, clientId, accountId, paymentAmount, paymentAddress, paymentDescription, callbackUrl, creationTime, expirationTime, status FROM invoices WHERE status IN (?, ?)", InvoiceStatusCreated, InvoiceStatusPending)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var invoice Invoice
		rows.Scan(&invoice.Id, &invoice.ClientId, &invoice.AccountId, &invoice.PaymentAmount, &invoice.PaymentAddress, &invoice.PaymentDescription, &invoice.CallbackUrl, &invoice.CreationTime, &invoice.ExpirationTime, &invoice.Status)
		invoices = append(invoices, invoice)
	}

	return invoices, nil
}

func FetchLRUWalletAddress() (string, error) {
	dbConnection := GetConnection()

	// Safety
	dbTx, err := dbConnection.Begin()
	if err != nil {
		return "", err
	}

	// Fetch LRU address
	var address string
	if err := dbTx.QueryRow("SELECT address FROM walletAddresses WHERE inUse = 0 ORDER BY lastUsed ASC LIMIT 1 FOR UPDATE").Scan(&address); err != nil {
		dbTx.Rollback()
		return "", err
	}

	// Lock LRU address
	if _, err := dbTx.Exec("UPDATE walletAddresses SET inUse = 1, lastUsed = NOW() WHERE address = ?", address); err != nil {
		dbTx.Rollback()
		return "", err
	}

	// Persist
	dbTx.Commit()

	return address, nil
}

func ReleaseLRUWalletAddress(address string) error {
	dbConnection := GetConnection()

	// Release LRU address
	if _, err := dbConnection.Exec("UPDATE walletAddresses SET inUse = 0, lastUsed = NOW() WHERE address = ?", address); err != nil {
		return err
	}

	return nil
}

func FetchWalletTransactionsByInvoiceId(invoiceId string) ([]WalletTransaction, error) {
	var walletTransactions []WalletTransaction
	dbConnection := GetConnection()
	rows, err := dbConnection.Query("SELECT id, invoiceId, walletAddress, paymentAmount, confirmationTime, discoveryTime FROM walletTransactions WHERE invoiceId = ?", invoiceId)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var walletTransaction WalletTransaction
		rows.Scan(&walletTransaction.Id, &walletTransaction.InvoiceId, &walletTransaction.WalletAddress, &walletTransaction.PaymentAmount, &walletTransaction.ConfirmationTime, &walletTransaction.DiscoveryTime)
		walletTransactions = append(walletTransactions, walletTransaction)
	}

	return walletTransactions, nil
}

func FetchPaymentAmountSumForInvoiceId(invoiceId string) (uint64, error) {
	var paymentAmountSum uint64
	dbConnection := GetConnection()
	if err := dbConnection.QueryRow("SELECT SUM(paymentAmount) FROM walletTransactions WHERE invoiceId = ?", invoiceId).Scan(&paymentAmountSum); err != nil {
		return paymentAmountSum, err
	}
	return paymentAmountSum, nil
}

func FetchPendingCallbacks() ([]Callback, error) {
	var callbacks []Callback
	dbConnection := GetConnection()
	rows, err := dbConnection.Query("SELECT id, invoiceId, requestTime, nextReqTime, reqErrors, status FROM callbacks WHERE status IN (?) AND nextReqTime < NOW()", CallbackStatusCreated)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var callback Callback
		rows.Scan(&callback.Id, &callback.InvoiceId, &callback.RequestTime, &callback.NextReqTime, &callback.ReqErrors, &callback.Status)
		callbacks = append(callbacks, callback)
	}

	return callbacks, nil
}
