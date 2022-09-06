package miner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/LambdaIM/proofDP"
	"github.com/LambdaIM/proofDP/math"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log"
	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/repo/fsrepo"
	"github.com/minio/highwayhash"
	mbase "github.com/multiformats/go-multibase"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"go.uber.org/atomic"
	"hash"
	"io"
	"io/fs"
	"io/ioutil"
	"math/big"
	mrand "math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	_1G      = int64(1 << 30)
	Segments = int64(1024 * 4)
	log      = logging.Logger("core/miner")
	doing    = []byte("0s")
	done     = []byte("1s")
)

type Miner struct {
	ctx         context.Context
	cfg         config.Config
	miningPath  []string
	running     *atomic.Bool
	lastTrigger *atomic.Time
	ticker      *time.Ticker
	miningDB    *leveldb.DB
	cli         *cli
}

func NewMiner(ctx context.Context, cfg config.Config) (miner *Miner, err error) {
	miner = &Miner{
		ctx:         ctx,
		cfg:         cfg,
		running:     atomic.NewBool(false),
		lastTrigger: atomic.NewTime(time.Now()),
		miningPath:  make([]string, 0),
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
	spec := miner.cfg.Datastore.Spec
	mounts, ok := spec["mounts"].([]interface{})
	if !ok {
		log.Fatalf("miner can not find any spec which named mounts")
	}
	for _, iface := range mounts {
		cfg, ok := iface.(map[string]interface{})
		if !ok {
			log.Fatalf("expect map for mountpoint")
		}
		childCfg, ok := cfg["child"]
		if !ok {
			log.Fatalf("expect child for mounts")
		}
		childCfgMap, ok := childCfg.(map[string]interface{})
		if !ok {
			log.Fatalf("expect map for `child` which was defined in `mounts` ")
		}
		childTyp, ok := childCfgMap["type"]
		if !ok {
			log.Warnf("expect map for `type` which was defined in `child` ")
			continue
		}
		childTypStr, ok := childTyp.(string)
		if !ok {
			log.Warnf("expect child type would be string %s", childTyp)
			continue
		}
		if childTypStr != "flatfs" {
			log.Warnf("unsupport type %s", childTypStr)
			continue
		}
		mountpoint := cfg["mountpoint"].(string)
		knownPath, err := fsrepo.BestKnownPath()
		if err != nil {
			log.Fatalf("while get ipfs path %s", err)
		}
		miner.miningPath = append(miner.miningPath, knownPath+mountpoint)
		log.Infof("mining on path %s", knownPath+mountpoint)
	}
	duration := time.Minute * time.Duration(miner.cfg.Miner.Delay)
	miner.ticker = time.NewTicker(duration)
	go miner.start()
	log.Infof("☀️  miner is setting up")
	return miner, nil
}

func (m *Miner) start() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case t := <-m.ticker.C:
			if m.blocking() {
				log.Warnf("mining task was still running which had started at %v, but now %v", m.lastTrigger.Load(), t)
				continue
			} else {
				log.Infof("🚀 minning")
				go m.mining()
			}
		}
	}
}

var (
	shaPool = sync.Pool{
		New: func() interface{} {
			return sha256.New()
		},
	}
	salt = [32]byte{1, 9, 0, 0, 1, 9, 9, 0, 2, 0, 0, 0, 2, 0, 1, 0, 2, 0, 1, 9, 2, 0, 2, 0, 2, 0, 2, 1, 2, 0, 2, 2}
)

func getHasher() hash.Hash {
	return shaPool.Get().(hash.Hash)
}

func putHasher(h hash.Hash) {
	h.Reset()
	shaPool.Put(h)
}

func hashSum(b []byte) []byte {
	hasher := getHasher()
	defer putHasher(hasher)
	hasher.Write(b)
	return hasher.Sum(nil)
}

func (m *Miner) getCid(cid string) ([]byte, error) {
	key := m.buildCid(cid)
	return m.miningDB.Get([]byte(key), &opt.ReadOptions{})
}

func (m *Miner) putCid(cid string, data []byte) error {
	key := m.buildCid(cid)
	return m.miningDB.Put([]byte(key), data, &opt.WriteOptions{})
}

func (m *Miner) buildCid(cid string) string {
	key := fmt.Sprintf("%s/%s", cid, hex.EncodeToString(hashSum(append([]byte(cid), salt[:]...))))
	return key
}

func (m *Miner) blocking() bool {
	return m.running.Load()
}

func (m *Miner) mining() {
	log.Infof("⛏️ DangDangDang")
	startTime := time.Now()
	defer func() {
		log.Infof("🧘 having a rest")
		m.running.Store(false)
		m.lastTrigger.Store(startTime)
	}()
	if !m.running.CAS(false, true) {
		log.Warnf("busy minning")
		return
	}
	var (
		accFiles []string
		accCids  []string
		acc      int64
		wait     sync.WaitGroup
		l        sync.RWMutex
	)
	wait.Add(len(m.miningPath))
	for _, baseDir := range m.miningPath {
		fileInfos, err := ioutil.ReadDir(baseDir)
		if err != nil {
			log.Errorf("while opening %s failed, %s", baseDir, err)
			return
		}
		log.Infof("search %s 🔍 %d branch", baseDir, len(fileInfos))
		go func(fs []fs.FileInfo, baseDir string) {
			defer wait.Done()
			sort.Sort(readyToS(fs))
			for _, dir := range fs {
				l.RLock()
				ret := acc > _1G
				l.RUnlock()
				if ret {
					return
				}
				//log.Infof("#### %s", dir.Name())
				if !dir.IsDir() {
					continue
				}
				if strings.Index(dir.Name(), "temp") != -1 {
					continue
				}
				parentName := dir.Name()
				parentDir := filepath.Join(baseDir, parentName)
				infos, err := ioutil.ReadDir(parentDir)
				if err != nil {
					log.Errorf("while indexing %s: %s", parentDir, err)
					return
				}
				sort.Sort(readyToS(infos))
				//log.Infof("#### search %s 🔍, len(%d)", parentDir, len(infos))
				for _, f := range infos {
					if f.IsDir() {
						continue
					}
					name := f.Name()
					idx := strings.Index(name, ".data")
					if idx == -1 {
						log.Warnf("skip path %s", name)
						continue
					}
					offset := len(name[:idx]) - 3
					lastTwo := name[offset : offset+2]
					if lastTwo != parentName {
						log.Warnf("skip path %s", filepath.Join(parentDir, name))
						continue
					}
					_, bs, derr := mbase.Decode("B" + name[:idx])
					if derr != nil {
						log.Warnf("while indexing %s : %s", name, err)
						continue
					}
					_, c, cerr := cid.CidFromBytes(bs)
					if cerr != nil {
						log.Warnf("while indexing %s: %s", name, err)
						continue
					}
					if f.Size() <= 0 {
						log.Warnf("while get zero size file %s", name)
						continue
					}
					cidStr := c.String()
					d, errd := m.getCid(cidStr)
					if errd != nil && !errors.Is(errd, leveldb.ErrNotFound) {
						log.Errorf("while recording progress : %s", errd)
						return
					}
					//log.Infof("## found name %s value %s", name, string(d))
					if len(d) != 0 && bytes.Equal(d, done) {
						//log.Infof("##### %s had been finished", name)
						continue
					}
					var (
						exceed  bool
						preStat bool
					)
					l.Lock()
					preStat = acc < _1G
					acc += f.Size()
					exceed = acc > _1G
					l.Unlock()
					if preStat {
						fullPath := filepath.Join(parentDir, name)
						accFiles = append(accFiles, fullPath)
						//log.Infof("### addd %s", fullPath)
						accCids = append(accCids, cidStr)
						err = m.putCid(cidStr, doing)
					}
					if exceed {
						log.Infof("🍔 enough material")
						return
					}
					if err != nil {
						log.Errorf("while record on object: %s", err)
					}
				}
			}
		}(fileInfos, baseDir)
	}
	wait.Wait()
	if acc < _1G {
		log.Infof("insufficient files 🛑")
		return
	}
	m.buildSector(accFiles, accCids)
}

func (m *Miner) buildSector(files []string, cids []string) {
	log.Infof("building sector 🚧")
	tmpp := filepath.Join(m.cfg.Miner.SealPath, fmt.Sprintf(".%d%d%d", os.Getpid(), time.Now().Unix(), mrand.Int()))
	file, err := os.OpenFile(tmpp, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		log.Errorf("while building sector: %s 🛑", err)
		return
	}
	defer file.Close()
	log.Infof("create temp file %s", file.Name())
	buffer := make([]byte, 2048)
	sha := sha256.New()
	dst := io.MultiWriter(sha, file)
	for _, f := range files {
		from, err := os.OpenFile(f, os.O_RDONLY, 0666)
		if err != nil {
			log.Errorf("while drain data from %s: %s 🛑", f, err)
			return
		}
		_ = file.Sync()
		stat, err := file.Stat()
		if err != nil {
			from.Close()
			log.Errorf("while getting stat from %s: %s", f, err)
			return
		}
		_, err = io.CopyBuffer(dst, io.LimitReader(from, _1G-stat.Size()), buffer)
		if err != nil {
			from.Close()
			log.Errorf("while building sector %s", err)
			return
		}
		from.Close()
	}
	hexSector := hex.EncodeToString(sha.Sum(nil))
	tof := filepath.Join(m.cfg.Miner.SealPath, hexSector)
	if err = os.Rename(tmpp, tof); err != nil {
		log.Errorf("while building sector %s: %s", tof, err)
		return
	}
	sectorF, err := os.OpenFile(tof, os.O_RDONLY, 0666)
	if err != nil {
		log.Errorf("while openning sector: %s", err)
		return
	}
	defer sectorF.Close()

	if err := m.proof(sectorF); err == nil {
		retryCommit(3, func() error {
			txn, err := m.miningDB.OpenTransaction()
			if err != nil {
				return err
			}
			for _, c := range cids {
				// log.Infof("## commiting %s", c)
				if err := txn.Put([]byte(m.buildCid(c)), done, &opt.WriteOptions{}); err != nil {
					return err
				}
			}
			return txn.Commit()
		})
	} else {
		log.Errorf("while proving %s", err)
	}

}

func retryCommit(n int, f func() error) {
	for i := 0; i < n; i++ {
		if f() == nil {
			break
		}
	}
}

func (m *Miner) proof(f *os.File) error {
	log.Infof("⚡ making proof")
	stat, err := f.Stat()
	if err != nil {
		return err
	}
	if stat.Size() < _1G {
		return errors.New(fmt.Sprintf("invalid sector %s", f.Name()))
	}
	segmentLen := _1G / Segments
	buffer := make([]byte, segmentLen)
	proofs := make([]proofDP.Proof, Segments)
	chals := make([]proofDP.Chal, Segments)
	segs := make([]Content, Segments)
	pps := make([]proofDP.PublicParams, Segments)
	sha := sha256.New()
	for index := int64(0); index < Segments; index++ {
		offset := index * segmentLen
		if _, err = f.Seek(offset, io.SeekStart); err != nil {
			log.Errorf("while making a move on %s to %d: %s", f.Name(), offset, err)
			return err
		}
		n, err := f.ReadAt(buffer, offset)
		if err != nil && err != io.EOF {
			log.Errorf("while reading from sector: %s", err)
			return err
		}
		if int64(n) != segmentLen {
			log.Errorf("expect length of segment would be %d, but %d", segmentLen, n)
			return errors.New("segment had invalid length")
		}
		proof, chal, pp, err := m.sealOnePiece(m.cli.webCli.Eth.Address().Bytes(), index, buffer)
		if err != nil {
			log.Errorf("while proving: %s", err)
			return err
		}
		proofs[index] = proof
		chals[index] = chal
		pps[index] = pp
		_, _ = sha.Write(buffer)

		seg := &Segment{}
		CloneInto(buffer, seg)
		segs[index] = seg
	}
	log.Info("🚧 seal tags ")
	h := sha.Sum(nil)
	hexSector := hex.EncodeToString(h)
	expectName := f.Name()
	index := strings.LastIndex(f.Name(), "/")
	if index != -1 && index != len(expectName)-1 {
		expectName = expectName[index+1:]
	}
	if hexSector != expectName {
		log.Errorf("sector had been modified, expect %s but %s", expectName, hexSector)
		return err
	}

	tree, err := NewTree(segs)
	if err != nil {
		log.Errorf("while building tree, branch 2, len(%d): %s", Segments, err)
		return err
	}
	m.cli.webCli.Eth.Address()
	idx := highwayhash.Sum64(m.cli.webCli.Eth.Address().Bytes(), tree.MerkleRoot()) % uint64(Segments)
	//log.Debugf("select idx %d", idx)
	return m.submitProof(pps[idx].Marshal(), chals[idx].Marshal(), proofs[idx].Marshal(), tree.MerkleRoot())
}

func (m *Miner) submitProof(pp, chal, proof string, root []byte) error {
	contract, err := m.cli.webCli.Eth.NewContract(AbsSubmitStr, m.cfg.Miner.ContractAddr)
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
	log.Debugf("addr byte %s", m.cli.webCli.Eth.Address().Bytes())
	name, err := contract.Methods("submitProof").Inputs.Pack(pp, proof, hex.EncodeToString(root32[:]), chal)
	if err != nil {
		log.Errorf("while pack param: %s", err)
		return err
	}
	data = append(data, name...)
	tokenAddress := common.HexToAddress(m.cfg.Miner.ContractAddr)
	txHash, err := m.cli.webCli.Eth.SyncSendRawTransaction(
		tokenAddress,
		big.NewInt(0),
		1050000,
		m.cli.webCli.Utils.ToGWei(100),
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

type Segment struct {
	slice []byte
}

func CloneInto(bs []byte, s *Segment) {
	s.slice = append(s.slice, bs...)
}

func (s *Segment) CalculateHash() ([]byte, error) {
	hasher := getHasher()
	defer putHasher(hasher)
	hasher.Write(s.slice)
	return hasher.Sum(nil), nil
}

func (s *Segment) Equals(other Content) (bool, error) {
	segment, ok := other.(*Segment)
	if !ok {
		return false, errors.New("expect miner/Segment")
	}
	return bytes.Equal(s.slice, segment.slice), nil
}

func (m *Miner) sealOnePiece(secret []byte, index int64, data []byte) (proofDP.Proof, proofDP.Chal, proofDP.PublicParams, error) {
	privateKey, err := proofDP.GeneratePrivateParams(secret)
	if err != nil {
		log.Errorf("while sealing sector with private key: %s", err)
		return proofDP.Proof{}, proofDP.Chal{}, proofDP.PublicParams{}, err
	}
	u, err := math.RandEllipticPt()
	publicKey := privateKey.GeneratePublicParams(u)
	tag, err := proofDP.GenTag(privateKey, publicKey, index, bytes.NewReader(data))
	if err != nil {
		log.Errorf("while building tags: %s", err)
		return proofDP.Proof{}, proofDP.Chal{}, proofDP.PublicParams{}, err
	}
	chal, err := proofDP.GenChal(index)
	if err != nil {
		log.Errorf("while building challenge: %s", err)
		return proofDP.Proof{}, proofDP.Chal{}, proofDP.PublicParams{}, err
	}
	proof, err := proofDP.Prove(publicKey, chal, tag, bytes.NewReader(data))
	if err != nil {
		log.Errorf("while proving: %s", err)
		return proofDP.Proof{}, proofDP.Chal{}, proofDP.PublicParams{}, err
	}
	return proof, chal, *publicKey, nil
}

func (m *Miner) Close() error {
	return m.miningDB.Close()
}

type readyToS []fs.FileInfo

func (f readyToS) Len() int {
	return len(f)
}

func (f readyToS) Less(i, j int) bool {
	return strings.Compare(f[i].Name(), f[j].Name()) < 0
}

func (f readyToS) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}
