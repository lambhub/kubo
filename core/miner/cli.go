package miner

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chenzhijie/go-web3"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"strconv"
	"strings"
)

type cli struct {
	webCli *web3.Web3
}

func newCli(providerUrl, account string, chainId int64) (*cli, error) {
	webCli, err := web3.NewWeb3(providerUrl)
	if err != nil {
		return nil, err
	}
	webCli.Eth.SetChainId(chainId)
	if err = webCli.Eth.SetAccount(account); err != nil {
		return nil, err
	}
	return &cli{webCli}, nil
}

func (c *cli) getChallenge(addr string) (string, error) {
	contract, err := c.webCli.Eth.NewContract(AbsSubmitStr, addr)
	if err != nil {
		return "", err
	}
	call, err := contract.Call(c.webCli.Eth.Address(), "getSeed")
	if err != nil {
		log.Errorf("⛓️ getSidByIndex with BlockChain, error: %s", err)
		return "", err
	}
	challenge := fmt.Sprintf("%s", call)
	log.Infof("#### getChallenge %s", challenge)
	return challenge, nil
}

func (c *cli) setChallenge(challenge, addr string) error {
	contract, err := c.webCli.Eth.NewContract(AbsSubmitStr, addr)
	if err != nil {
		return err
	}
	mth := contract.Methods("setSeed")
	data := mth.ID
	name, err := contract.Methods("setSeed").Inputs.Pack(challenge)
	if err != nil {
		log.Errorf("while pack param: %s", err)
		return err
	}
	data = append(data, name...)
	tokenAddress := common.HexToAddress(addr)
	txHash, err := c.webCli.Eth.SyncSendRawTransaction(
		tokenAddress,
		big.NewInt(0),
		1050000,
		c.webCli.Utils.ToGWei(100),
		data,
	)
	if err != nil {
		log.Infof("⛓️ interact with BlockChain, error: %s", err)
	} else {
		marshal, err := json.Marshal(txHash)
		if err != nil {
			log.Infof("⛓️ interact with BlockChain, error: %s", err)
		}
		log.Infof("⛓ interact with BlockChain, respone: %s", string(marshal))
	}
	return err
}

func (c *cli) recordSidCids(sid string, cid []string, addr string) error {
	contract, err := c.webCli.Eth.NewContract(AbsSubmitStr, addr)
	if err != nil {
		return err
	}
	mth := contract.Methods("buildSidCidsMap")
	data := mth.ID
	name, err := contract.Methods("buildSidCidsMap").Inputs.Pack(sid, cid)
	if err != nil {
		log.Errorf("while pack param: %s", err)
		return err
	}
	data = append(data, name...)
	tokenAddress := common.HexToAddress(addr)
	txHash, err := c.webCli.Eth.SyncSendRawTransaction(
		tokenAddress,
		big.NewInt(0),
		1050000,
		c.webCli.Utils.ToGWei(100),
		data,
	)
	if err != nil {
		log.Infof("⛓️ interact with BlockChain, error: %s", err)
	} else {
		marshal, err := json.Marshal(txHash)
		if err != nil {
			log.Infof("⛓️ interact with BlockChain, error: %s", err)
		}
		log.Infof("⛓ interact with BlockChain, respone: %s", string(marshal))
	}
	return err
}

func (c *cli) getSid(idx uint64, addr string) (string, error) {
	contract, err := c.webCli.Eth.NewContract(AbsSubmitStr, addr)
	if err != nil {
		return "", err
	}
	bIdx := big.NewInt(int64(idx))
	call, err := contract.Call(c.webCli.Eth.Address(), "getSidByIndex", bIdx)
	if err != nil {
		log.Errorf("⛓️ getSidByIndex with BlockChain, error: %s", err)
		return "", err
	}
	sid := fmt.Sprintf("%s", call)
	//log.Infof("#### getSidByIndex %s", sid)
	return sid, nil
}

func (c *cli) getMaxLen(addr string) (uint64, error) {
	contract, err := c.webCli.Eth.NewContract(AbsSubmitStr, addr)
	if err != nil {
		return 0, err
	}
	call, err := contract.Call(c.webCli.Eth.Address(), "getSidsCount")
	if err != nil {
		log.Errorf("⛓️ getMaxLength with BlockChain, error: %s", err)
		return 0, err
	}
	raw := fmt.Sprintf("%s", call)
	//log.Infof("##### getSidsCount %s", raw)
	l, err := strconv.Atoi(raw)
	return uint64(l), err
}

func (c *cli) canBuildSector(addr string) (bool, error) {
	contract, err := c.webCli.Eth.NewContract(AbsSubmitStr, addr)
	if err != nil {
		return false, err
	}
	call, err := contract.Call(c.webCli.Eth.Address(), "getSectorRole")
	if err != nil {
		log.Errorf("⛓️ getSectorRole with BlockChain, error: %s", err)
		return false, err
	}
	raw := strings.TrimSpace(fmt.Sprintf("%s", call))
	//log.Infof("##### getSectorRole %s", raw)
	if raw == "1" {
		log.Infof("building sector ....")
	} else {
		log.Infof("minning sector ....")
	}
	return raw == "1", nil
}

func (c *cli) PaddingAddress() [32]byte {
	var ans [32]byte
	address := c.webCli.Eth.Address()
	idx := len(address) - 1
	for i := 31; i >= 0; i-- {
		if idx < 0 {
			ans[i] = 0
		} else {
			ans[i] = address[idx]
		}
		idx--
	}
	return ans
}

func (c *cli) canSaveUp(contractAddr, cid string, size int64) (bool, error) {
	contract, err := c.webCli.Eth.NewContract(FilterAbI, contractAddr)
	if err != nil {
		return false, err
	}
	call, err := contract.Call(c.webCli.Eth.Address(), "getCidSize", cid)
	if err != nil {
		return false, err
	}
	raw := strings.TrimSpace(fmt.Sprintf("%s", call))
	log.Debugf("FilterContract Return %s", raw)
	b := strings.Contains(raw, "true")
	expectSize := uint64(0)
	useless := ""
	_, err = fmt.Sscanf(raw, "[%s %d]", &useless, &expectSize)
	if err != nil {
		return false, err
	}
	upper := int64(float64(expectSize) * 1.05)
	lower := int64(float64(expectSize) * 0.95)
	return b && lower <= size && size <= upper, nil
}

func leftPad(bs []byte) ([32]byte, error) {
	var ans [32]byte
	if len(bs) > 32 {
		return ans, errors.New("expect padding a byte array with len(32)")
	}
	idx := len(bs) - 1
	for i := 31; i >= 0; i-- {
		if idx < 0 {
			ans[i] = 0
		} else {
			ans[i] = bs[idx]
		}
		idx--
	}
	return ans, nil
}

const AbsSubmitStr = `[
	{
		"inputs": [
			{
				"internalType": "address payable",
				"name": "_proxy",
				"type": "address"
			},
			{
				"internalType": "address",
				"name": "_owner",
				"type": "address"
			}
		],
		"stateMutability": "nonpayable",
		"type": "constructor"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "timestamp",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "address",
				"name": "chalAddr",
				"type": "address"
			}
		],
		"name": "ChalAddrUpdated",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"internalType": "address",
				"name": "oldOwner",
				"type": "address"
			},
			{
				"indexed": false,
				"internalType": "address",
				"name": "newOwner",
				"type": "address"
			}
		],
		"name": "OwnerChanged",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"internalType": "address",
				"name": "newOwner",
				"type": "address"
			}
		],
		"name": "OwnerNominated",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"internalType": "address",
				"name": "proxyAddress",
				"type": "address"
			}
		],
		"name": "ProxyUpdated",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"internalType": "address",
				"name": "chalAddr",
				"type": "address"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "timestamp",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "currRound",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "string",
				"name": "seed",
				"type": "string"
			}
		],
		"name": "SeedUpdate",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "timestamp",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "address[]",
				"name": "newAddrs",
				"type": "address[]"
			},
			{
				"indexed": false,
				"internalType": "uint256[]",
				"name": "newShares",
				"type": "uint256[]"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "maxIndex",
				"type": "uint256"
			}
		],
		"name": "ShareUpdated",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "status",
				"type": "uint256"
			}
		],
		"name": "StatusUpdated",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"components": [
					{
						"internalType": "address",
						"name": "_addr",
						"type": "address"
					},
					{
						"internalType": "string",
						"name": "_sid",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "_pubkey",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "_sha256",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "_chal",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "_proof",
						"type": "string"
					}
				],
				"indexed": false,
				"internalType": "struct Pdp.PubkeyProof",
				"name": "",
				"type": "tuple"
			}
		],
		"name": "SubmitProofStruct",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "timestamp",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "address",
				"name": "verifyAddr",
				"type": "address"
			}
		],
		"name": "VerifyAddrUpdated",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"internalType": "address",
				"name": "verifyAddr",
				"type": "address"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "timestamp",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "currRound",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "status",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "address[]",
				"name": "minerAddr",
				"type": "address[]"
			}
		],
		"name": "VerifyResultUpdated",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"internalType": "address",
				"name": "gateway",
				"type": "address"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "timestamp",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "string",
				"name": "sid",
				"type": "string"
			},
			{
				"indexed": false,
				"internalType": "string[]",
				"name": "cids",
				"type": "string[]"
			}
		],
		"name": "buildSidCidsMapUpdated",
		"type": "event"
	},
	{
		"inputs": [],
		"name": "acceptOwnership",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "_addr",
				"type": "address"
			}
		],
		"name": "addSectorWhiteList",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "string",
				"name": "sid",
				"type": "string"
			},
			{
				"internalType": "string[]",
				"name": "cids",
				"type": "string[]"
			}
		],
		"name": "buildSidCidsMap",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "closeSubmit",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "destroyPdp",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "getAllShare",
		"outputs": [
			{
				"internalType": "address[]",
				"name": "",
				"type": "address[]"
			},
			{
				"internalType": "uint256[]",
				"name": "",
				"type": "uint256[]"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "getAmountRound",
		"outputs": [
			{
				"internalType": "uint256",
				"name": "",
				"type": "uint256"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "getBalance",
		"outputs": [
			{
				"internalType": "uint256",
				"name": "",
				"type": "uint256"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "getChalAddr",
		"outputs": [
			{
				"internalType": "address",
				"name": "",
				"type": "address"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "string",
				"name": "sid",
				"type": "string"
			}
		],
		"name": "getCidsBySid",
		"outputs": [
			{
				"internalType": "string[]",
				"name": "",
				"type": "string[]"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "getCurrRound",
		"outputs": [
			{
				"internalType": "uint256",
				"name": "",
				"type": "uint256"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "_round",
				"type": "uint256"
			}
		],
		"name": "getHistoryVerifyDataByRound",
		"outputs": [
			{
				"components": [
					{
						"internalType": "address",
						"name": "_addr",
						"type": "address"
					},
					{
						"internalType": "string",
						"name": "_sid",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "_pubkey",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "_sha256",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "_chal",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "_proof",
						"type": "string"
					}
				],
				"internalType": "struct Pdp.PubkeyProof[]",
				"name": "",
				"type": "tuple[]"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "getMaxIndex",
		"outputs": [
			{
				"internalType": "uint256",
				"name": "",
				"type": "uint256"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "getMsgSender",
		"outputs": [
			{
				"internalType": "address",
				"name": "",
				"type": "address"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "getSectorRole",
		"outputs": [
			{
				"internalType": "uint256",
				"name": "",
				"type": "uint256"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "_addr",
				"type": "address"
			}
		],
		"name": "getSectorRoleByAddr",
		"outputs": [
			{
				"internalType": "uint256",
				"name": "",
				"type": "uint256"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "getSeed",
		"outputs": [
			{
				"internalType": "string",
				"name": "",
				"type": "string"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "_addr",
				"type": "address"
			}
		],
		"name": "getShareByAddr",
		"outputs": [
			{
				"internalType": "uint256",
				"name": "",
				"type": "uint256"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "index",
				"type": "uint256"
			}
		],
		"name": "getSidByIndex",
		"outputs": [
			{
				"internalType": "string",
				"name": "",
				"type": "string"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "getSidsCount",
		"outputs": [
			{
				"internalType": "uint256",
				"name": "",
				"type": "uint256"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "getStatus",
		"outputs": [
			{
				"internalType": "uint256",
				"name": "",
				"type": "uint256"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "getVerifyAddr",
		"outputs": [
			{
				"internalType": "address",
				"name": "",
				"type": "address"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "getVerifyData",
		"outputs": [
			{
				"components": [
					{
						"internalType": "address",
						"name": "_addr",
						"type": "address"
					},
					{
						"internalType": "string",
						"name": "_sid",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "_pubkey",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "_sha256",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "_chal",
						"type": "string"
					},
					{
						"internalType": "string",
						"name": "_proof",
						"type": "string"
					}
				],
				"internalType": "struct Pdp.PubkeyProof[]",
				"name": "",
				"type": "tuple[]"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "messageSender",
		"outputs": [
			{
				"internalType": "address",
				"name": "",
				"type": "address"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "_owner",
				"type": "address"
			}
		],
		"name": "nominateNewOwner",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "nominatedOwner",
		"outputs": [
			{
				"internalType": "address",
				"name": "",
				"type": "address"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "openSubmit",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "owner",
		"outputs": [
			{
				"internalType": "address",
				"name": "",
				"type": "address"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "proxy",
		"outputs": [
			{
				"internalType": "contract Proxy",
				"name": "",
				"type": "address"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "_amount",
				"type": "uint256"
			}
		],
		"name": "setAmountRound",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "_rate",
				"type": "uint256"
			}
		],
		"name": "setBaseRate",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "_addr",
				"type": "address"
			}
		],
		"name": "setChalAddr",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "sender",
				"type": "address"
			}
		],
		"name": "setMessageSender",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address payable",
				"name": "_proxy",
				"type": "address"
			}
		],
		"name": "setProxy",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "string",
				"name": "_seed",
				"type": "string"
			}
		],
		"name": "setSeed",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "_addr",
				"type": "address"
			}
		],
		"name": "setVerifyAddr",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "_addr",
				"type": "address"
			}
		],
		"name": "subSectorWhiteList",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "string",
				"name": "_sid",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "_pubkey",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "_proof",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "_sha256",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "_chal",
				"type": "string"
			}
		],
		"name": "submitProof",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address[]",
				"name": "_addrs",
				"type": "address[]"
			},
			{
				"internalType": "uint256[]",
				"name": "_shares",
				"type": "uint256[]"
			}
		],
		"name": "updateShareWhiteList",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address[]",
				"name": "_addrArr",
				"type": "address[]"
			}
		],
		"name": "verifyResult",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"stateMutability": "payable",
		"type": "receive"
	}
]`

const FilterAbI = `
[
    {
      "inputs": [
        {
          "internalType": "string",
          "name": "cid",
          "type": "string"
        }
      ],
      "name": "getCidSize",
      "outputs": [
        {
          "internalType": "bool",
          "name": "",
          "type": "bool"
        },
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    }
 ]
`
