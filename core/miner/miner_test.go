package miner

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"github.com/LambdaIM/proofDP"
	"github.com/LambdaIM/proofDP/math"
	"github.com/chenzhijie/go-web3"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
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

func getRandSecret() []byte {
	res := make([]byte, 32)
	_, err := rand.Read(res)
	if err != nil {
		panic(err)
	}
	return res
}

func TestF(t *testing.T) {
	file := make([]byte, 1024)
	_, err := rand.Read(file)
	require.Nil(t, err)
	secret := getRandSecret()
	sp, err := proofDP.GeneratePrivateParams(secret)
	assert.NoError(t, err)

	u, err := math.RandEllipticPt()
	assert.NoError(t, err)
	pp := sp.GeneratePublicParams(u)

	tag, err := proofDP.GenTag(sp, pp, int64(3), bytes.NewReader(file))
	assert.NoError(t, err)
	fmt.Println(tag)
	marshal := tag.Marshal()
	fmt.Println(len([]byte(marshal)))
}

func TestCrc(t *testing.T) {
	a := "QmTDwxrphm5abp18LRmM5v8jimw3RW5BXnBD1tFf42Yav7"
	//fmt.Println(len(a))
	//_, bs, err := mbase.Decode(a)
	//fmt.Println(err)
	//_, c, err := cid.CidFromBytes(bs)
	//fmt.Println(err)
	//fmt.Println(c)
	decode, err := cid.Decode(a)
	fmt.Println(err)
	fmt.Println(decode)

}
