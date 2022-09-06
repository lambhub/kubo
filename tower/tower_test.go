package tower

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test123(t *testing.T) {

	b1, err := hex.DecodeString("B83e4338AD5eA6Fc0b20a952c8CaDD857349d6F6")
	require.Nil(t, err)
	fmt.Println(b1)
	b2, err := hex.DecodeString("b83e4338ad5ea6fc0b20a952c8cadd857349d6f6")
	require.Nil(t, err)
	fmt.Println(b2)
	fmt.Println(bytes.Equal(b1, b2))
}
