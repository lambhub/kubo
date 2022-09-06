package miner

import (
	"errors"
	"github.com/chenzhijie/go-web3"
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
				"internalType": "address",
				"name": "_verifyAddr",
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
				"indexed": true,
				"internalType": "address",
				"name": "oldOwner",
				"type": "address"
			},
			{
				"indexed": true,
				"internalType": "address",
				"name": "newOwner",
				"type": "address"
			}
		],
		"name": "OwnerSet",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "",
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
				"internalType": "address",
				"name": "",
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
				"internalType": "uint256",
				"name": "",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "",
				"type": "uint256"
			}
		],
		"name": "VerifyResultUpdated",
		"type": "event"
	},
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "_addr",
				"type": "address"
			},
			{
				"internalType": "uint256",
				"name": "_share",
				"type": "uint256"
			}
		],
		"name": "addWhiteList",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "address",
				"name": "newOwner",
				"type": "address"
			}
		],
		"name": "changeOwner",
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
		"name": "getOwner",
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
		"name": "getTotalShare",
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
		"name": "openSubmit",
		"outputs": [],
		"stateMutability": "nonpayable",
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
		"name": "subWhiteList",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
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
