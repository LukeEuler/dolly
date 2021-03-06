package common

import (
	"testing"
)

func TestCut(t *testing.T) {
	type args struct {
		raw           string
		decimalPoints uint
		tailPoints    uint
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"test 1",
			args{"1234", 2, 1},
			"12.3",
			false,
		},
		{
			"test 2",
			args{"13a4", 7, 3},
			"",
			true,
		},
		{
			"test 3",
			args{"1234", 7, 3},
			"0",
			false,
		}, {
			"test 4",
			args{"1234", 7, 5},
			"0.00012",
			false,
		}, {
			"test 5",
			args{"1234", 2, 0},
			"12",
			false,
		}, {
			"test 6",
			args{"1234", 0, 0},
			"1234",
			false,
		}, {
			"test 7",
			args{"-1234", 2, 1},
			"-12.3",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Cut(tt.args.raw, tt.args.decimalPoints, tt.args.tailPoints)
			if (err != nil) != tt.wantErr {
				t.Errorf("Cut() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Cut() = %v, want %v", got, tt.want)
			}
		})
	}
}
