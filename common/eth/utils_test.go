package eth

import (
	"reflect"
	"testing"
)

func TestTopicDataToAddress(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{
			"test 1",
			"0x00000000000000000000000032a8984b8565f7aa996aa265eebbb91b0b4dabb7",
			"32a8984b8565f7aa996aa265eebbb91b0b4dabb7",
			false,
		},
		{
			"test 2",
			"0x000000000000000000000000002a8984b8565f7aa996aa265eebbb91b0b4dabb7",
			"02a8984b8565f7aa996aa265eebbb91b0b4dabb7",
			false,
		},
		{
			"test 3",
			"0x000000000000000000000000000a8984b8565f7aa996aa265eebbb91b0b4dabb7",
			"00a8984b8565f7aa996aa265eebbb91b0b4dabb7",
			false,
		},
		{
			"test 4",
			"0x00000000000000000000000032a8984b8565f7aa996aa265eebbb91b0b4dabb0",
			"32a8984b8565f7aa996aa265eebbb91b0b4dabb0",
			false,
		},
		{
			"test 5",
			"0x00000000000000000000000032a8984b8565f7aa996aa265eebbb91b0b4dab00",
			"32a8984b8565f7aa996aa265eebbb91b0b4dab00",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TopicDataToAddress(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("TopicDataToAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("TopicDataToAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTopicDataToBigInt(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{
			"test 1",
			"0x0000000000000000000000000000000067ae7714f0ef463d8ae14dd2ecb568c0",
			"137816358485103147079446902817395402944",
			false,
		},
		{
			"test 1",
			"0x00000000000000000000000000000000000000000000000000000000000f4240",
			"1000000",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TopicDataToBigInt(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("TopicDataToBigInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.String(), tt.want) {
				t.Errorf("TopicDataToBigInt() = %v, want %v", got, tt.want)
			}
		})
	}
}
