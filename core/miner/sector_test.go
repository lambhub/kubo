package miner

import (
	"fmt"
	"github.com/ipfs/kubo/proofDP"
	"github.com/ipfs/kubo/proofDP/math"
	"github.com/stretchr/testify/require"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

func createf(base, name string, size int, t *testing.T) (string, []byte) {
	p := path.Join(base, name)
	n := make([]byte, size)
	_, err := rand.Read(n)
	require.Nil(t, err)
	return p, n
}

func suit(t *testing.T) ([]string, [][]byte) {
	var (
		ps []string
		fs [][]byte
	)
	tempDir := os.TempDir()
	a := path.Join(tempDir, "a")
	aa2 := path.Join(tempDir, "a2")
	//
	b := path.Join(a, "b")
	//
	c := path.Join(a, "c")
	//
	bb2 := path.Join(aa2, "bb2")
	// 4M
	cc1, cc1f := createf(bb2, "cc1", 4*(1<<20), t)
	ps = append(ps, cc1)
	fs = append(fs, cc1f)
	//
	a1, a1f := createf(a, "a1", 4*(1<<20), t)
	ps = append(ps, a1)
	fs = append(fs, a1f)
	//// 4M
	//a2, a2f := createf(a, "a2", 4*(1<<20), t)
	//ps = append(ps, a2)
	//fs = append(fs, a2f)
	// b
	b1, b1f := createf(b, "b1", 4*(1<<20), t)
	ps = append(ps, b1)
	fs = append(fs, b1f)
	b2, b2f := createf(b, "b2", 4*(1<<20), t)
	ps = append(ps, b2)
	fs = append(fs, b2f)
	b3, b3f := createf(b, "b3", 4*(1<<20), t)
	ps = append(ps, b3)
	fs = append(fs, b3f)
	b4, b4f := createf(b, "b4", 4*(1<<20), t)
	ps = append(ps, b4)
	fs = append(fs, b4f)
	c1, c1f := createf(c, "c1", 4*(1<<20), t)
	ps = append(ps, c1)
	fs = append(fs, c1f)

	c2, c2f := createf(c, "c2", 4*(1<<20)-23, t)
	ps = append(ps, c2)
	fs = append(fs, c2f)

	return ps, fs
}

func TestSector_StepByStep(t *testing.T) {
	sts, fs := suit(t)
	n := make([]byte, 32)
	_, err := rand.Read(n)
	require.Nil(t, err)
	privateKey, err := proofDP.GeneratePrivateParams(n)
	require.Nil(t, err)
	u, err := math.RandEllipticPt()
	publicKey := privateKey.GeneratePublicParams(u)
	sector := NewSector(privateKey, publicKey)
	var finished bool
	for idx, s := range sts {
		prefix := os.TempDir()
		tt := strings.TrimPrefix(s, prefix)
		cids := strings.Split(tt, string(filepath.Separator))
		finished, err = sector.StepByStep(cids, fs[idx])
		require.Nil(t, err)
	}
	require.Equal(t, uint32(23), sector.padding)
	require.True(t, finished)
}

func Test(t *testing.T) {
	a := "QmVp1AYHHGyrKw7SAzWkrmcb3yXUP9FJuzNhbK3ihiq3pPBcuhDmfuLPd9riqyjwIPwu5xvbw6NRpEwrTKykFe+2SYmYadqwC0s66+6V6K5zff6rt6Fv5bPWjWKsXf2Gd1M2RqwIVLhghcA8KKC0+ee1j3sOLEZKdV94bP53PMY5crQyvKoPgnfRT4AfOpGbD+IUhdXJ1QINCuak4SBkQPd8Y=,UmlOl4LpuxAyzmY5wlAyU1hxgKnJfDaL6QOpGRjkL5k93RdI7GbMNKu5lzHiPalIprkt7LLzYM21eKJzHQDXVnXA8kOGdQhmmQuQMkviJDFPX6kotjMZHppqNJOkQLnCorL/P+is67+pux5lbKkNb7uhXmDZzxMStcveCrGGkVw=,LxLWW66cECEHDEK/Jo82NsWOnUiwzOCu2zKZZlb7pTIZmpDDfPxZIRBKGI4pMCfHJoSES5WXjanUTZQPmq4guircILwJjUn8yu1DFl4NNbsGHfMld9AJDX9kG2SH1bgz7qVwaLEa0slboe99gCIzSQdDji9CFAs5l6734AFUj/Y=Xrl/V+r9gLdMj2i90VrZ4Y9jP4Y=,nA5abiI3ZQ5yYibff2K363HcXcuDFK47Yd5CfjlvOr0fYW2FeXTTfU4tKvHnxkVIfwvxP0Gh6klB5/5ERNJsc4E+Lf8G12sGB6pGzjH0x/c4RUEUnJli5YRecNJ/KkwojFO4PTgjMyTFyRfp1q/NtkdxTqZeJ1vt7jaCumx7tCU=,VJcZMMNJTtOoVcsdhkeaVE502an9u/6e5KAvyT5KbSsDQHSokNxn+cZoPQc9CwTFtAm/dQuOY9ZwmQeqIOVcWBGUvDi0f5hU1XPfpzQNtnpNpilMFdW34Jrd2b7GdIEYoaOqnfI4lmMlJzphaeMvgD9C4WYRIWFVakjwLMCFfV4=37202961d893f0d35516c976590b14eb0292c8808d7bd90264f3f15c44d54540NDg=,KRR3PN6/nXeI6wYXlA75NcUHWAk="

	fmt.Println(len([]byte(a)))
}
