package tower

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/LambdaIM/proofDP"
	"github.com/chenzhijie/go-web3"
	"github.com/ethereum/go-ethereum/common"
	logging "github.com/ipfs/go-log"
	"github.com/ipfs/kubo/core/miner"
	"github.com/minio/highwayhash"
	"go.uber.org/atomic"
	"math/big"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var log = logging.Logger("tower")

type Tower struct {
	RemoteUrl    string
	ContractAddr string
	t            *time.Ticker
	ctx          context.Context
	running      *atomic.Bool
	lastTrigger  *atomic.Time
	concurrent   int
	web3         *web3.Web3
}

func NewTower(ctx context.Context, url string, d time.Duration, chainID int64, privateKey, contractAddr string) (*Tower, error) {
	logging.SetDebugLogging()
	web3cli, err := web3.NewWeb3(url)
	if err != nil {
		log.Errorf("while building web3 client: %s", err)
		return nil, err
	}
	web3cli.Eth.SetChainId(chainID)
	if err = web3cli.Eth.SetAccount(privateKey); err != nil {
		log.Errorf("while setting private key which must be hex format: %s", err)
		return nil, err
	}
	t := &Tower{
		ctx:          ctx,
		RemoteUrl:    url,
		t:            time.NewTicker(d),
		running:      atomic.NewBool(false),
		lastTrigger:  atomic.NewTime(time.Now()),
		concurrent:   runtime.NumCPU(),
		web3:         web3cli,
		ContractAddr: contractAddr,
	}
	return t, nil
}

func (t *Tower) isRunning() bool {
	return t.running.Load()
}

func (t *Tower) Run() {
	log.Infof("tower is running on address %s, contract %s !!", t.web3.Eth.Address().Hex(), t.ContractAddr)
	for {
		select {
		case <-t.ctx.Done():
			return
		case _ = <-t.t.C:
			if t.isRunning() {
				log.Warnf("task is running until now, last time: %v", t.lastTrigger.Load())
			} else {
				log.Infof("🚀 start verification")
				go t.stepVerify()
			}
		}
	}
}

type Ctx struct {
	PP      proofDP.PublicParams
	Chal    proofDP.Chal
	Proof   proofDP.Proof
	Address common.Address
	Sha     [32]byte
	Idx     uint64
}

func (t *Tower) fetchTask() ([]Ctx, error) {
	contract, err := t.web3.Eth.NewContract(miner.AbsSubmitStr, t.ContractAddr)
	if err != nil {
		log.Errorf("while web3 new contract %s", err)
		return nil, err
	}
	call, err := contract.Call(t.web3.Eth.Address(), "getVerifyData")
	if err != nil {
		log.Infof("⛓️ interact with BlockChain, error: %s", err)
	}
	var strs []string
	sprintf := fmt.Sprintf("%s", call)
	st := 0
	left := -1
	log.Infof("hard code respone")
	for idx, c := range sprintf {
		if c == '{' && idx != len(sprintf)-1 {
			left = idx + 1
			st++
		}
		if c == '}' && st == 1 && left < idx {
			strs = append(strs, sprintf[left:idx])
			st--
		}
	}
	var ans []Ctx
	log.Infof("⛓ interact with BlockChain, respone: %v", call)
	// 0xB83e4338AD5eA6Fc0b20a952c8CaDD857349d6F6
	// N0jjBs21nA/GqAlrBgwBwZ51d++HhQk/2qAtnWIaQh0fni91DoViuTZb//tLEdstLk9E4lhtFxDe0EfyllamTZ3jo88ULBrLjZ76hai2N+X8p7l1Tr+FTC53vAiIIu8vWosXRM2dOH6owZ8e+njAFSvj8/Mo9rItwuP4ee8c8r4=,lOxdk8NPdzEZgUjUgqL6bfgrIdEsvujXEn+7UhPIb/w6+QKNJJ1moX81SrHr9RCGySJHnzpr6j6gm++XmblDnF23IXF2Q1DtqiMcwBukyVBA3JRKQbq2O7G6k0kyjqxiYsvOyIq1q/6AbB1lGnuAcdd+h5dOm4dBuTUlKkWyu+E=,UyuxJO8IAvIinazQcqosU/i7OiK1KTj2bvwEL4cMn2Z2P72lKxybSm9ajICMEguGPjBQcZHfiScDKrlA54wASpGH02Z/03Yx/RDZjqLJRD/xykn5JYtPf4thhoF9ME3o/UL2RSth6cpza9ExAuOBRdaVRRH3kl0s78HTmqgWlbY=
	// 4b1659fe622a85adc54398bbc1cf8f0863deed9a7de600bd72bfd1293335d0d6
	// NzE0,P8FXoACMXL4v/aE7VNwi+6VkcDg=
	// EJQPhhpywLi4B11loJqCejIE3jY=,CMQzLmLBnnj2VZzG/yN+Kg5F5oSZ7EYWWiIaj6MgzA7Npfnp680Aw4KIkb3/u5P87ClmZG6sWyXQ2vANC+wkDR3SDBqOiLb0MTq8a/HDnlyvl6tFwW3/wq33pX5BkzxCelS35QG1iJbOOhpethLsUjm8Jc1icJ4HO1dOBXI59ik=,M9LDi0vIQ2PMbAgT8/COjhFTVVcgc3ZUInK1WXTSm2kY8SmLF0+Yf0+iFrfnzZAY2bcLwWjvnyKJylhUrRyL2h7lhL9SrZgSzAbgXapej/Z+uxpUizLEmg0d4kp8vf0XmGZvynXYAoL3ybhPLecfuPsQFFu0Lbclo4PRxRPl/OE=
	for _, str := range strs {
		split := strings.SplitN(str, " ", 5)
		if len(split) != 5 {
			return nil, errors.New("expect message with 5 space")
		}
		// ==== 0xB83e4338AD5eA6Fc0b20a952c8CaDD857349d6F6 => b83e4338ad5ea6fc0b20a952c8cadd857349d6f6
		address := common.HexToAddress(split[0])
		log.Debugf("addr byte %v", address)
		publicParams, err := proofDP.ParsePublicParams(split[1])
		if err != nil {
			log.Errorf("while parse pp %s: %s", split[1], err)
			return nil, err
		}
		var root [32]byte
		log.Debugf("root hex %s", split[2])
		nroot, err := hex.DecodeString(split[2])
		if err != nil {
			log.Errorf("while parse sha256 %s: %s", split[2], err)
			return nil, err
		}
		copy(root[:], nroot[:32])
		chal, err := proofDP.ParseChal(split[3])
		if err != nil {
			log.Errorf("while parse chal %s: %s", split[3], err)
			return nil, err
		}
		chalRaw := strings.SplitN(split[3], ",", 2)
		if len(chalRaw) != 2 {
			log.Errorf("while parse raw chal with `,` %s: %s", split[3], err)
			return nil, errors.New("invalid challenge")
		}
		numInStr, err := base64.StdEncoding.DecodeString(chalRaw[0])
		if err != nil {
			log.Errorf("while decode base64 %s: %s", chalRaw[0], err)
			return nil, err
		}
		idx, err := strconv.ParseUint(string(numInStr), 10, 64)
		if err != nil {
			log.Errorf("while str to num %s: %s", string(numInStr), err)
			return nil, err
		}
		proof, err := proofDP.ParseProof(split[4])
		if err != nil {
			log.Errorf("while parse proof %s: %s", split[4], err)
			return nil, err
		}
		ans = append(ans, Ctx{
			Address: address,
			PP:      *publicParams,
			Chal:    chal,
			Proof:   proof,
			Sha:     root,
			Idx:     idx,
		})
	}
	log.Infof("finished hard code return value")
	return ans, nil
}

func (t *Tower) verifyCallback(set map[common.Address]struct{}) error {
	var ans [][20]byte
	for k, _ := range set {
		ans = append(ans, k)
	}
	contract, err := t.web3.Eth.NewContract(miner.AbsSubmitStr, t.ContractAddr)
	if err != nil {
		log.Errorf("while web3 new contract %s", err)
		return err
	}
	tokenAddress := common.HexToAddress(t.ContractAddr)
	mth := contract.Methods("verifyResult")
	data := mth.ID
	name, err := contract.Methods("verifyResult").Inputs.Pack(ans)
	if err != nil {
		log.Errorf("while pack param: %s", err)
		return err
	}
	data = append(data, name...)
	txHash, err := t.web3.Eth.SyncSendRawTransaction(
		tokenAddress,
		big.NewInt(0),
		1000000,
		t.web3.Utils.ToGWei(100),
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

func (t *Tower) stepVerify() {
	log.Infof("⛏️ DangDangDang")
	startTime := time.Now()
	defer func() {
		log.Infof("🧘 having a rest")
		t.running.Store(false)
		t.lastTrigger.Store(startTime)
	}()
	if !t.running.CAS(false, true) {
		log.Warnf("busy verifying")
		return
	}
	tasks, err := t.fetchTask()
	if err != nil {
		log.Errorf("while fetching tasks: %s", err)
		return
	}
	chunkSize := (len(tasks) + t.concurrent - 1) / t.concurrent
	log.Infof("🔭 verify len(%d), chunk len(%d)", len(tasks), chunkSize)
	var (
		set = make(map[common.Address]struct{})
		w   sync.WaitGroup
		m   sync.Mutex
	)
	for i := 0; i < len(tasks); i += chunkSize {
		end := i + chunkSize
		if end > len(tasks) {
			end = len(tasks)
			chunkSize = len(tasks) - i
		}
		w.Add(chunkSize)
		for j := i; j < end; j++ {
			target := tasks[j]
			go func(ctx Ctx) {
				defer w.Done()
				log.Infof("verify %s", hex.EncodeToString(ctx.Address[:]))
				idx := highwayhash.Sum64(ctx.Address.Bytes(), ctx.Sha[:]) % uint64(miner.Segments)
				if idx != ctx.Idx {
					log.Errorf("while verify index, expect: %d, but: %d", idx, ctx.Idx)
					return
				}
				if proofDP.VerifyProof(&ctx.PP, ctx.Chal, ctx.Proof) {
					log.Infof("success %s", common.BytesToAddress(ctx.Address[:]).Hex())
					m.Lock()
					set[ctx.Address] = struct{}{}
					m.Unlock()
				}
			}(target)
		}
		w.Wait()
	}
	if err = t.closeSubmit(); err != nil {
		log.Errorf("while close submit: %s", err)
		return
	}
	if err := t.verifyCallback(set); err != nil {
		log.Errorf("while call verify method: %s", err)
	} else if len(set) == 0 {
		log.Infof("🧘 None tasks")
	}
}

func (t *Tower) closeSubmit() error {
	contract, err := t.web3.Eth.NewContract(miner.AbsSubmitStr, t.ContractAddr)
	if err != nil {
		log.Errorf("while web3 new contract %s", err)
		return err
	}
	// ================================================
	nmths := contract.Methods("closeSubmit")
	ndata := nmths.ID
	tokenAddress := common.HexToAddress(t.ContractAddr)
	_, err = t.web3.Eth.SyncSendRawTransaction(
		tokenAddress,
		big.NewInt(0),
		1000000,
		t.web3.Utils.ToGWei(100),
		ndata,
	)
	if err != nil {
		log.Errorf("while calling closeSubmit: %s", err)
		return err
	}
	return nil
}
