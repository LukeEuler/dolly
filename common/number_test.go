package common

import (
	"math/big"
	"reflect"
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

func TestStringToBigInt(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    *big.Int
		wantErr bool
	}{
		{
			"test 1",
			"123",
			big.NewInt(123),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StringToBigInt(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("StringToBigInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StringToBigInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHexStringToBigInt(t *testing.T) {
	tests := []struct {
		name    string
		content string
		value   string
		wantErr bool
	}{
		{
			"test 1",
			"1a",
			"26",
			false,
		},
		{
			"test 1",
			"0000000000000000000000000000000000000000000001032c50d3b4c90a0000",
			"4780900000000000000000",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotB, err := HexStringToBigInt(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("HexStringToBigInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotB.String(), tt.value) {
				t.Errorf("HexStringToBigInt() = %v, want %v", gotB, tt.value)
			}
		})
	}
}

func TestFloatStringToBigInt(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		decimalPoints uint
		want          *big.Int
		wantErr       bool
	}{
		{
			"test 1",
			"123.456",
			4,
			big.NewInt(1234560),
			false,
		},
		{
			"test 2",
			"123.456",
			2,
			big.NewInt(12345),
			false,
		},
		{
			"test 3",
			"123456",
			2,
			big.NewInt(12345600),
			false,
		},
		{
			"test 4",
			"123.456",
			0,
			big.NewInt(123),
			false,
		},
		{
			"test 5",
			"-123.456",
			2,
			big.NewInt(-12345),
			false,
		},
		{
			"test 6",
			"+123.456",
			2,
			nil,
			true,
		},
		{
			"test 7",
			"1.",
			2,
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FloatStringToBigInt(tt.content, tt.decimalPoints)
			if (err != nil) != tt.wantErr {
				t.Errorf("FloatStringToBigInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FloatStringToBigInt() = %v, want %v", got, tt.want)
			}
		})
	}
}
