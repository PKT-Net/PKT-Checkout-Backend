package wallet

type BlockchainTransaction struct {
	Id            string
	WalletAddress string
	PaymentAmount uint64
	DiscoveryTime uint64
	Confirmations uint32
}
