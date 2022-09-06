package commands

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	cmds "github.com/ipfs/go-ipfs-cmds"
	"github.com/mitchellh/go-homedir"
	"os"
)

var Keygen = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "swarm key for network",
	},
	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		oldPath, err := homedir.Expand("~/.ipfs/swarm.key")
		if err != nil {
			return res.Emit(fmt.Sprintf("while expand path for `~/.ipfs/swarm.key`: %s", err))
		}
		if _, err := os.Stat(oldPath); err == nil {
			// path/to/whatever exists
			return res.Emit("`~/.ipfs/swarm.key` already exists")
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return res.Emit(fmt.Sprintf("while detectiving `~/.ipfs/swarm.key` : %s", err))
		}
		file, err := os.OpenFile(oldPath, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			return res.Emit(fmt.Sprintf("while openning path for `~/.ipfs/swarm.key`: %s", err))
		}
		key := make([]byte, 32)
		_, err = rand.Read(key)
		if err != nil {
			return res.Emit(fmt.Sprintf("While trying to read random source: %s", err))
		}
		content := "/key/swarm/psk/1.0.0/\n/base16/\n" + hex.EncodeToString(key)
		_, err = file.WriteString(content)
		if err != nil {
			return res.Emit(fmt.Sprintf("while generating swarm:%s", err))
		}
		return nil
	},
}
