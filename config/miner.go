package config

type Miner struct {
	RemoteURL          string
	Delay              uint
	Record             string
	SealPath           string
	PrivateKey         string
	ContractAddr       string
	FilterContractAddr string
	ChainID            int64
}
