package miner

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/chenzhijie/go-web3"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

func Test123(t *testing.T) {
	webCli, err := web3.NewWeb3("http://18.143.13.243:8545")
	require.Nil(t, err)
	webCli.Eth.SetChainId(92001)
	err = webCli.Eth.SetAccount("pk")
	require.Nil(t, err)
	contract, err := webCli.Eth.NewContract(AbsSubmitStr, "0xf5f85683a6008341Af6aaD02371197897C18AB0E")
	require.Nil(t, err)
	//pad, err := leftPad([]byte{1, 2, 3})
	//require.Nil(t, err)
	c, err := contract.Methods("setName").Inputs.Pack("ppp", webCli.Utils.ToWei(0.2))
	fmt.Println(c)
	fmt.Println(err)
}

func Test456(t *testing.T) {
	client, err := ethclient.Dial("http://18.143.13.243:8545")
	require.Nil(t, err)
	defer client.Close()
	privateKey, err := crypto.HexToECDSA("pk")
	require.Nil(t, err)
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal(err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(92001))
	require.Nil(t, err)
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)     // in wei
	auth.GasLimit = uint64(300000) // in units
	auth.GasPrice = gasPrice

}
