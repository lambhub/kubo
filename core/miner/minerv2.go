package miner

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/proofDP"
	"github.com/ipfs/kubo/proofDP/math"
	"github.com/minio/highwayhash"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"go.uber.org/atomic"
	"io"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const miningIdx = "v2/mining/sector/idx"

type V2 struct {
	ctx         context.Context
	cfg         config.Config
	miningPath  []string
	running     *atomic.Bool
	lastTrigger *atomic.Time
	ticker      *time.Ticker
	miningDB    *leveldb.DB
	cli         *cli
	ipfs        iface.CoreAPI
	buildSector bool
}

func NewMinerV2(ctx context.Context, cfg config.Config, api iface.CoreAPI) (miner *V2, err error) {
	miner = &V2{
		ctx:         ctx,
		cfg:         cfg,
		running:     atomic.NewBool(false),
		lastTrigger: atomic.NewTime(time.Now()),
		miningPath:  make([]string, 0),
		ipfs:        api,
	}
	miner.cli, err = newCli(miner.cfg.Miner.RemoteURL, miner.cfg.Miner.PrivateKey, miner.cfg.Miner.ChainID)
	if err != nil {
		log.Debugf("while init web3: %s", err)
		log.Warn("⚠️ miner offline")
		return nil, nil
	}
	if err = os.MkdirAll(miner.cfg.Miner.Record, 0777); err != nil {
		log.Errorf("while mkdir record: %s", err)
		return nil, err
	}
	if err = os.MkdirAll(miner.cfg.Miner.SealPath, 0777); err != nil {
		log.Errorf("while mkdir seal: %s", err)
		return nil, err
	}
	datastore, err := leveldb.OpenFile(miner.cfg.Miner.Record, &opt.Options{
		Compression: opt.SnappyCompression,
	})
	if err != nil {
		if strings.Contains(fmt.Sprintf("%s", err), "resource temporarily unavailable") {
			log.Warn("⚠️ miner offline")
			return nil, nil
		}
		log.Errorf("while open record db: %s", err)
		return nil, err
	}
	miner.miningDB = datastore
	canBuildSector, err := miner.cli.canBuildSector(miner.cfg.Miner.ContractAddr)
	if err != nil {
		log.Errorf("while building miner's client %s", err)
	}
	miner.buildSector = canBuildSector
	duration := time.Minute * time.Duration(miner.cfg.Miner.Delay)
	miner.ticker = time.NewTicker(duration)
	return miner, nil
}

func (v *V2) blocking() bool {
	return v.running.Load()
}

func (v *V2) Start() {
	for {
		select {
		case <-v.ctx.Done():
			return
		case t := <-v.ticker.C:
			if v.blocking() {
				log.Warnf("mining task was still running which had started at %v, but now %v", v.lastTrigger.Load(), t)
				continue
			} else {
				log.Infof("🚀 Work, work.")
				if v.buildSector {
					go v.building()
				} else {
					go v.mining()
				}
			}
		}
	}
}

func (v *V2) mining() {
	log.Infof("⛏️ DangDangDang")
	startTime := time.Now()
	defer func() {
		log.Infof("🧘 having a rest")
		v.running.Store(false)
		v.lastTrigger.Store(startTime)
	}()
	if !v.running.CAS(false, true) {
		log.Warnf("busy minning")
		return
	}
	val, err := v.miningDB.Get([]byte(miningIdx), &opt.ReadOptions{})
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		log.Errorf("while get mining cnt record: %s", err)
		return
	}
	idx := uint64(0)
	nextIdx := idx
	if len(val) == 8 {
		idx = binary.LittleEndian.Uint64(val)
		nextIdx = idx + 1
	}
	maxLen, err := v.cli.getMaxLen(v.cfg.Miner.ContractAddr)
	if err != nil {
		log.Errorf("while get max index: %s", err)
		return
	}
	if maxLen <= nextIdx {
		log.Infof("there isn't any new sector")
		return
	}
	challenge, err := v.cli.getChallenge(v.cfg.Miner.ContractAddr)
	if err != nil {
		log.Errorf("while getting challenge %s", err)
		return
	}
	sid, err := v.cli.getSid(nextIdx, v.cfg.Miner.ContractAddr)
	if err != nil {
		log.Errorf("while get sid: %s", err)
		return
	}
	log.Infof("downloading %s ...", sid)
	f, err := v.ipfs.Unixfs().Get(v.ctx, path.New(sid))
	if err != nil {
		log.Errorf("while download sid %s: %s", sid, err)
		return
	}
	var file files.File
	switch f := f.(type) {
	case files.File:
		file = f
	case files.Directory:
		return
	default:
		return
	}
	fsize, err := file.Size()
	if err != nil {
		log.Errorf("while sizing sector: %s", err)
		return
	}
	dat, err := io.ReadAll(io.LimitReader(file, fsize))
	if err != nil {
		log.Errorf("while read sector: %s", err)
		return
	}
	var s Sector
	err = s.UnmarshalBinary(dat)
	if err != nil {
		log.Errorf("while getting sector %s: %s", sid, err)
		return
	}

	publicParams := s.pp
	proofs := make([]proofDP.Proof, SegmentNum)
	chals := make([]proofDP.Chal, SegmentNum)
	for i := 0; i < SegmentNum; i++ {
		tagStr := s.tags[i]
		tag, err := proofDP.ParseTag(tagStr)
		if err != nil {
			log.Errorf("while parsing tag: %s", err)
			return
		}
		seg := s.segments[i*segmentLen : (i+1)*segmentLen]
		chal, err := proofDP.GenChallengeWithN(int64(i), challenge)
		if err != nil {
			log.Errorf("while building challenge: %s", err)
			return
		}
		proof, err := proofDP.Prove(publicParams, chal, tag, bytes.NewReader(seg))
		if err != nil {
			log.Errorf("while proving: %s", err)
			return
		}
		proofs[i] = proof
		chals[i] = chal
	}
	target := highwayhash.Sum64(v.cli.webCli.Eth.Address().Bytes(), s.merkleRoot) % uint64(SegmentNum)
	log.Infof("submitting proof ....")

	err = v.submitProof(sid, publicParams.Marshal(), chals[target].Marshal(), proofs[target].Marshal(), s.merkleRoot)
	if err != nil {
		log.Errorf("while submit proof %s", err)
	} else {
		var nb [8]byte
		binary.LittleEndian.PutUint64(nb[:], nextIdx)
		//log.Infof("#### record index %d", nextIdx)
		retryCommit(100, func() error {
			return v.miningDB.Put([]byte(miningIdx), nb[:], &opt.WriteOptions{})
		})
	}
}

func (v *V2) filter() error {
	rootDir := v.cfg.Miner.SealPath
	if rootDir[len(rootDir)-1] == filepath.Separator {
		rootDir = rootDir[:len(rootDir)-1]
	}
	dir, err := os.ReadDir(rootDir)
	if err != nil {
		return err
	}
	for _, f := range dir {
		fullp := filepath.Join(rootDir, f.Name())
		size, err := DirSize(fullp)
		if err != nil {
			return err
		}
		canSaveUp, err := v.cli.canSaveUp(v.cfg.Miner.FilterContractAddr, f.Name(), size)
		if err != nil {
			return err
		}
		if !canSaveUp {
			err := v.putCid(f.Name(), done)
			_ = os.RemoveAll(fullp)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func DirSize(path string) (int64, error) {
	log.Debugf("DirSizing %s", path)
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

func (v *V2) building() {
	log.Infof("🏗️️ building")
	startTime := time.Now()
	defer func() {
		log.Infof("🧘 having a rest")
		v.running.Store(false)
		v.lastTrigger.Store(startTime)
	}()
	if !v.running.CAS(false, true) {
		log.Warnf("busy minning")
		return
	}

	err := v.filter()
	if err != nil {
		log.Errorf("while filtering cid-files: %s", err)
		return
	}

	pri, pub, err := v.randParam(v.cli.webCli.Eth.Address().Bytes())
	if err != nil {
		log.Errorf("while rand pdp params: %s", err)
		return
	}
	newSector := func() (*Sector, []string, map[string]struct{}) {
		s := NewSector(&pri, &pub)
		return &s, make([]string, 0), make(map[string]struct{})
	}
	sector, cidsBundle, cidRoots := newSector()
	rootDir := v.cfg.Miner.SealPath
	if rootDir[len(rootDir)-1] == filepath.Separator {
		rootDir = rootDir[:len(rootDir)-1]
	}
	status := false
	err = filepath.Walk(rootDir, func(path string, info fs.FileInfo, err error) error {
		if status {
			return nil
		}
		log.Debugf("#### under %s", path)
		log.Debugf("### %d %d", len(cidsBundle), len(cidRoots))
		if errors.Is(err, filepath.SkipDir) {
			return nil
		} else if err != nil {
			return err
		}
		if path != rootDir && !isCID(info.Name()) {
			return filepath.SkipDir
		}
		if info.IsDir() {
			bs, err := v.getCid(info.Name())
			if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
				log.Errorf("while recording progress : %s", err)
				return err
			}
			if len(bs) != 0 && bytes.Equal(bs, done) {
				//log.Infof("##### %s had been finished", path)
				return filepath.SkipDir
			}
			return nil
		}
		if info.Size() > int64(1<<25) {
			return nil
		}
		t := strings.TrimPrefix(path, rootDir+string(filepath.Separator))
		cids := strings.Split(t, string(filepath.Separator))
		if len(cids) == 0 {
			log.Errorf("while indexing path: invalid cids length")
			return errors.New("invalid cids length")
		}
		//log.Debugf("####### path route %+v", cids)
		bs, err := v.getCid(info.Name())
		if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
			log.Errorf("while recording progress : %s", err)
			return err
		}
		//log.Infof("## found name %s value %s", info.Name(), path)
		if len(bs) != 0 && bytes.Equal(bs, done) {
			//log.Infof("##### %s had been finished", path)
			return nil
		}
		cidsBundle = append(cidsBundle, info.Name())
		dat, err := os.ReadFile(path)
		if err != nil {
			log.Errorf("while reading %s: %s", path, err)
			return err
		}
		finish, err := sector.StepByStep(cids, dat)
		log.Infof("sector seg %d %t", sector.DataLen(), finish)
		if err != nil && errors.Is(err, ErrNotEnoughFreeSize) {
			log.Infof("skip %s temporarily ", info.Name())
			return nil
		}
		if err != nil {
			return err
		}
		//log.Infof("#### step %s ", info.Name())
		err = v.putCid(info.Name(), doing)
		if err != nil {
			log.Errorf("while doing cid %s: %s", info.Name(), err)
			return err
		}
		cidRoots[cids[0]] = struct{}{}
		if finish { // reset new sector anyway
			defer func() {
				sector, cidsBundle, cidRoots = newSector()
			}()
			binaryDat, err := sector.MarshalBinary()
			if err != nil {
				log.Errorf("while making sector: %s", err)
				return err
			}
			unixfs := v.ipfs.Unixfs()
			resolved, err := unixfs.Add(v.ctx, files.NewBytesFile(binaryDat))
			if err != nil {
				log.Errorf("while upload sector: %s", err)
				return err
			}
			// @TODO: upload sector cid => cid1, cid2....
			sectorCid := resolved.Cid().String()
			rootCids := make([]string, 0)
			for k := range cidRoots {
				rootCids = append(rootCids, k)
			}
			if err = v.cli.recordSidCids(sectorCid, rootCids, v.cfg.Miner.ContractAddr); err != nil {
				log.Errorf("while upload sector id: %s", err)
				return err
			}
			// commit cids
			retryCommit(5, func() error {
				txn, err := v.miningDB.OpenTransaction()
				if err != nil {
					return err
				}
				for _, c := range cidsBundle {
					//log.Infof("## commiting %s", c)
					if err := txn.Put([]byte(v.buildCid(c)), done, &opt.WriteOptions{}); err != nil {
						return err
					}
				}
				return txn.Commit()
			})
			log.Infof("%s build completed", sectorCid)
			status = true
		}
		return nil
	})
	if err != nil {
		log.Errorf("Error: %s", err)
	}
}

func (v *V2) randParam(secret []byte) (proofDP.PrivateParams, proofDP.PublicParams, error) {
	privateKey, err := proofDP.GeneratePrivateParams(secret)
	if err != nil {
		return proofDP.PrivateParams{}, proofDP.PublicParams{}, err
	}
	u, err := math.RandEllipticPt()
	publicKey := privateKey.GeneratePublicParams(u)
	return *privateKey, *publicKey, nil
}

func (v *V2) getCid(cid string) ([]byte, error) {
	key := v.buildCid(cid)
	return v.miningDB.Get([]byte(key), &opt.ReadOptions{})
}

func (v *V2) putCid(cid string, data []byte) error {
	key := v.buildCid(cid)
	return v.miningDB.Put([]byte(key), data, &opt.WriteOptions{})
}

func (v *V2) buildCid(cid string) string {
	key := fmt.Sprintf("%s/%s", cid, hex.EncodeToString(hashSum(append([]byte(cid), saltV2[:]...))))
	return key
}

func (v *V2) submitProof(sid, pp, chal, proof string, root []byte) error {
	contract, err := v.cli.webCli.Eth.NewContract(AbsSubmitStr, v.cfg.Miner.ContractAddr)
	if err != nil {
		log.Errorf("while init contract: %s", err)
		return err
	}
	root32, err := leftPad(root)
	if err != nil {
		log.Errorf("while padding root: %s", err)
		return err
	}
	mth := contract.Methods("submitProof")
	data := mth.ID
	//log.Debugf("root hex %s", hex.EncodeToString(root32[:]))
	//log.Debugf("addr byte %s", v.cli.webCli.Eth.Address().Bytes())
	//log.Infof("##### pp %s", pp)
	//log.Infof("##### proof %s", proof)
	//log.Infof("##### sha %s", hex.EncodeToString(root32[:]))
	//log.Infof("##### chal %s", chal)
	name, err := contract.Methods("submitProof").Inputs.Pack(sid, pp, proof, hex.EncodeToString(root32[:]), chal)
	if err != nil {
		log.Errorf("while pack param: %s", err)
		return err
	}
	data = append(data, name...)
	tokenAddress := common.HexToAddress(v.cfg.Miner.ContractAddr)
	txHash, err := v.cli.webCli.Eth.SyncSendRawTransaction(
		tokenAddress,
		big.NewInt(0),
		1500000,
		v.cli.webCli.Utils.ToGWei(50),
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

func isCID(name string) bool {
	c, err := cid.Decode(name)
	if err != nil {
		log.Warnf("while decode cid %s : %s", name, err)
		return false
	}
	return c.String() == name
}
