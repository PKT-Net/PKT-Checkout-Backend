package wallet

import (
	"pkt-checkout/database"
	"time"

	"github.com/google/uuid"
)

func (s *Server) Scan() {
	// Fetch invoices that require wallet backend scanning
	invoices, err := database.FetchPendingInvoices()
	if err != nil {
		return
	}

	// Fetch transactions from wallet backend
	wbTransactions, err := s.getTransactions()
	if err != nil {
		return
	}

	// Update states on invoices as necessary
	for _, invoice := range invoices {
		// Fetch transactions from database
		dbTransactions, err := database.FetchWalletTransactionsByInvoiceId(invoice.Id)
		if err != nil {
			continue
		}

		// Filter transactions from wallet backend
		var invoiceWbTransactions []BlockchainTransaction
		for _, tx := range wbTransactions {
			if invoice.PaymentAddress == tx.WalletAddress {
				invoiceWbTransactions = append(invoiceWbTransactions, tx)
			}
		}

		// Invoice has definitely expired
		if invoice.ExpirationTime.Before(time.Now()) && len(dbTransactions) == 0 && len(invoiceWbTransactions) == 0 {
			invoice.Status = "expired"
			invoice.Update()

			database.ReleaseLRUWalletAddress(invoice.PaymentAddress)

			// Request callback
			if len(invoice.CallbackUrl) > 0 {
				var callback database.Callback
				callback.Id = uuid.New().String()
				callback.InvoiceId = invoice.Id
				callback.RequestTime = time.Now()
				callback.NextReqTime = time.Now()
				callback.ReqErrors = 0
				callback.Status = database.CallbackStatusCreated
				callback.Save()
			}

			continue
		}

		// Invoice has received at least one transaction
		if invoice.Status == "created" && len(invoiceWbTransactions) > 0 {
			invoice.Status = "pending"
			invoice.Update()
		}

		// Persist wallet backend transactions
		for _, tx := range invoiceWbTransactions {
			persistTx := true
			for _, txDb := range dbTransactions {
				if tx.Id == txDb.Id {
					persistTx = false
					break
				}
			}
			if persistTx {
				if tx.Confirmations >= s.TxConfirmations {
					var walletTransaction database.WalletTransaction
					walletTransaction.Id = tx.Id
					walletTransaction.InvoiceId = invoice.Id
					walletTransaction.WalletAddress = tx.WalletAddress
					walletTransaction.PaymentAmount = tx.PaymentAmount
					walletTransaction.DiscoveryTime = time.Unix(int64(tx.DiscoveryTime), 0)
					walletTransaction.ConfirmationTime = time.Now()
					walletTransaction.Save()
				}
			}
		}

		// Fetch the sum of all payments made towards the invoice
		paymentAmountSum, err := database.FetchPaymentAmountSumForInvoiceId(invoice.Id)
		if err != nil {
			continue
		}

		// Invoice may have been paid at this point
		if paymentAmountSum >= invoice.PaymentAmount {
			invoice.Status = "paid"
			invoice.Update()

			database.ReleaseLRUWalletAddress(invoice.PaymentAddress)

			// Request callback
			if len(invoice.CallbackUrl) > 0 {
				var callback database.Callback
				callback.Id = uuid.New().String()
				callback.InvoiceId = invoice.Id
				callback.RequestTime = time.Now()
				callback.NextReqTime = time.Now()
				callback.ReqErrors = 0
				callback.Status = database.CallbackStatusCreated
				callback.Save()
			}

			continue
		}
	}
}
