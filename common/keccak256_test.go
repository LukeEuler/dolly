package common

import "testing"

func TestKeccak256String(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			"test 1",
			"Transfer(address,address,uint256)",
			"ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
		},
		{
			"test 2",
			"multiTransfer(((address,address,uint256,uint256)[]),bytes)",
			"1e6ae9e4a9ba6ded9a63b3fe906865f94098714f6cf75922c997ddfa7ad6b138",
		},
		{
			"test 3",
			"DepositNativeAsset(address,address,uint256)",
			"4cd95e3231f8db03da63cd445f41dca23adb082131f15cd7cd76b34cefe7e708",
		},
		{
			"test 4",
			"TransferETH(address,uint256)",
			"fd69c215b8b91dab5e96ff0bcbaf5dc372919948eea2003ae16481c036f816f8",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Keccak256String(tt.content); got != tt.want {
				t.Errorf("Keccak256String() = %v, want %v", got, tt.want)
			}
		})
	}
}
