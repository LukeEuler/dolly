package rpc

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParams(t *testing.T) {
	type tempStruct struct {
		Params any `json:"params,omitempty"`
	}

	axx := Params()
	bs, err := json.Marshal(tempStruct{Params: axx})
	assert.NoError(t, err)
	assert.Equal(t, "{}", string(bs))

	axx = Params(nil)
	bs, err = json.Marshal(tempStruct{Params: axx})
	assert.NoError(t, err)
	assert.Equal(t, `{"params":[null]}`, string(bs))

	axx = Params(emptyStruct{})
	bs, err = json.Marshal(tempStruct{Params: axx})
	assert.NoError(t, err)
	assert.Equal(t, `{"params":{}}`, string(bs))

	axx = Params([]bool{})
	bs, err = json.Marshal(tempStruct{Params: axx})
	assert.NoError(t, err)
	assert.Equal(t, `{"params":[]}`, string(bs))

	axx = Params(3, "abc")
	bs, err = json.Marshal(tempStruct{Params: axx})
	assert.NoError(t, err)
	assert.Equal(t, `{"params":[3,"abc"]}`, string(bs))
}
