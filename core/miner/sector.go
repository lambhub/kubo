package miner

import (
	"bytes"
	"errors"
	"github.com/ipfs/kubo/core/util"
	"github.com/ipfs/kubo/proofDP"
)

var (
	v1 = []byte("v1")
)

const (
	sectorLen  = 1 << 25
	SegmentNum = 64
	segmentLen = sectorLen / SegmentNum
)

type Sector struct {
	sp         *proofDP.PrivateParams
	pp         *proofDP.PublicParams
	nextWrite  int
	tags       []string
	segments   []byte
	padding    uint32
	forest     util.Forest
	merkleRoot []byte
}

func NewSector(sp *proofDP.PrivateParams, pp *proofDP.PublicParams) Sector {
	return Sector{
		sp:         sp,
		pp:         pp,
		nextWrite:  0,
		tags:       make([]string, SegmentNum),
		segments:   make([]byte, sectorLen),
		padding:    sectorLen,
		merkleRoot: nil,
		forest: util.Forest{
			Trees: make([]*util.Entry, 0),
		},
	}
}

func (s *Sector) DataLen() int {
	return s.nextWrite
}

func (s *Sector) StepByStep(pathRoute []string, dat []byte) (bool, error) {
	if err := s.stepBuilding(pathRoute, dat); err != nil {
		//log.Errorf("while building sector: %s", err)
		return false, err
	}
	ready, err := s.stepSetup()
	if err != nil {
		log.Errorf("while setting up sector: %s", err)
		return false, err
	}
	if !ready {
		//log.Debugf("insufficient files 🛑")
		return false, nil
	}

	if err := s.stepMerkleTree(); err != nil {
		log.Errorf("while setting up sector: %s", err)
		return false, err
	}
	return true, nil
}

func (s *Sector) canWrite(dat []byte) bool {
	return sectorLen-s.nextWrite >= len(dat)
}

func (s *Sector) stepBuilding(pathRoute []string, dat []byte) error {
	if len(pathRoute) == 0 {
		return errors.New("expect a node with path route")
	}
	offset, err := s.write(dat)
	if err != nil {
		return err
	}
	return s.bindPath(pathRoute, offset, len(dat))
}

// `find root node in forest and then insert node with full path` Or
// `find root node and then find leaf node`
func (s *Sector) bindPath(pathRoute []string, offset, length int) error {
	var rootNode *util.Entry
	for _, root := range s.forest.Trees {
		if root.Cid == pathRoute[0] {
			rootNode = root
			break
		}
	}
	if rootNode == nil {
		rootNode = &util.Entry{
			Cid:      pathRoute[0],
			Typ:      util.EntryTyp_Dir,
			Children: make([]*util.Link, 0),
		}
		s.forest.Trees = append(s.forest.Trees, rootNode)
	} else if entry := util.Find(rootNode, pathRoute[len(pathRoute)-1]); entry != nil {
		return nil
	}
	pathRoute = pathRoute[1:]
	var ent *util.Entry
	if len(pathRoute) == 0 {
		rootNode.Typ = util.EntryTyp_File
		ent = rootNode
	} else {
		ent = util.InsertWithFullPath(rootNode, pathRoute)
	}
	if ent == nil {
		log.Errorf("insert path %v into %+v failed. ", pathRoute, rootNode)
		return errors.New("sector binding failed. ")
	}
	ent.Offset = uint32(offset)
	ent.Len = uint32(length)
	return nil
}

var ErrNotEnoughFreeSize = errors.New("sector hasn't enough free size")

func (s *Sector) write(dat []byte) (int, error) {
	if len(dat) > sectorLen {
		return 0, errors.New("length of data exceed sector's length")
	}
	if !s.canWrite(dat) {
		return 0, ErrNotEnoughFreeSize
	}
	offset := s.nextWrite
	copy(s.segments[s.nextWrite:], dat)
	s.nextWrite = s.nextWrite + len(dat)
	s.padding = uint32(sectorLen - s.nextWrite)
	return offset, nil
}

func (s *Sector) stepSetup() (bool, error) {
	if s.padding >= segmentLen { // a sector hasn't been done
		return false, nil
	}
	// padding
	s.segments = append(s.segments, make([]byte, s.padding)...)
	return true, s.setup()
}

func (s *Sector) setup() error {
	for i := 0; i < SegmentNum; i++ {
		tag, err := proofDP.GenTag(s.sp, s.pp, int64(i), bytes.NewReader(s.segments[i*segmentLen:(i+1)*segmentLen]))
		if err != nil {
			return err
		}
		s.tags[i] = tag.Marshal()
	}
	return nil
}

func (s *Sector) stepMerkleTree() error {
	segments := make([]Content, SegmentNum)
	for i := 0; i < SegmentNum; i++ {
		segments[i] = &Segment{
			slice: s.segments[i*segmentLen : (i+1)*segmentLen],
		}
	}
	tree, err := NewTree(segments)
	if err != nil {
		return err
	}
	s.merkleRoot = tree.MerkleRoot()
	return nil
}

func (s *Sector) MarshalBinary() (data []byte, err error) {
	sector := util.Sector{
		Tags: &util.Tags{
			Tags: s.tags,
		},
		Segments: s.segments,
		Root:     s.merkleRoot,
		Padding:  s.padding,
		F:        &s.forest,
		Pp:       s.pp.Marshal(),
	}
	data, err = sector.Marshal()
	if err != nil {
		return nil, err
	}
	return append(v1, data...), nil
}

func (s *Sector) UnmarshalBinary(data []byte) error {
	if len(data) <= len(v1) {
		return errors.New("expect a sector with spec version")
	}
	data = data[len(v1):]
	var ss util.Sector
	if err := ss.Unmarshal(data); err != nil {
		return err
	}
	s.tags = ss.Tags.Tags
	s.segments = ss.Segments
	s.merkleRoot = ss.Root
	s.padding = ss.Padding
	s.forest = *ss.F
	params, err := proofDP.ParsePublicParams(ss.Pp)
	if err != nil {
		return err
	}
	s.pp = params
	return nil
}
