package eth

import (
	"encoding/hex"
	"math/big"

	"github.com/pkg/errors"

	dc "github.com/LukeEuler/dolly/common"
)

func TopicDataToAddress(content string) (string, error) {
	if len(content) > 40 {
		content = content[len(content)-40:]
	}
	content = dc.FormatHexString(content)
	bs, err := hex.DecodeString(content)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return hex.EncodeToString(bs), nil
}

func TopicDataToBigInt(content string) (*big.Int, error) {
	raw := dc.FormatHexString(content)
	value, ok := big.NewInt(0).SetString(raw, 16)
	if !ok {
		return nil, errors.Errorf("can not parse %s as big.Int", content)
	}
	return value, nil
}
