package common

import (
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

var (
	errInvalidNum = func(content string) error {
		return errors.Errorf("invalid number string: [%s]", content)
	}
)

/*
Cut 将一个整数字符串，转化为浮点数字符串
decimalPoints： 整数转化为浮点数时，缩放的精度
tailPoints：整数转化为浮点数后，保留的小数位数

example:
(1234,2,1) -> 12.3
*/
func Cut(raw string, decimalPoints, tailPoints uint) (string, error) {
	neg := false
	if strings.HasPrefix(raw, "-") {
		raw = raw[1:]
		neg = true
	}
	if ok, _ := regexp.MatchString(`^(0|[1-9][0-9]*)$`, raw); !ok {
		return "", errInvalidNum(raw)
	}
	length := len(raw)
	if tailPoints > decimalPoints {
		tailPoints = decimalPoints
	}
	if length <= int(decimalPoints-tailPoints) {
		return "0", nil
	}
	raw = raw[:length-int(decimalPoints-tailPoints)]
	length = len(raw)
	head := "0"
	tail := raw
	if length > int(tailPoints) {
		head = raw[:length-int(tailPoints)]
		tail = raw[length-int(tailPoints):]
	}
	for i := length; i < int(tailPoints); i++ {
		tail = "0" + tail
	}
	if neg {
		head = "-" + head
	}
	if len(tail) == 0 {
		return head, nil
	}
	return head + "." + tail, nil
}

/*
FloatStringToBigInt 浮点数转化为整数
decimalPoints： 浮点数转化为整数时，缩放的精度

example:
(123.456, 2) -> 12345
*/
func FloatStringToBigInt(content string, decimalPoints uint) (*big.Int, error) {
	if ok, _ := regexp.MatchString(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`, content); !ok {
		return nil, errInvalidNum(content)
	}
	head := content
	tail := ""
	if strings.Contains(content, ".") {
		splitList := strings.Split(content, ".")
		if len(splitList) != 2 {
			return nil, errInvalidNum(content)
		}
		head, tail = splitList[0], splitList[1]
	}
	if len(tail) > int(decimalPoints) {
		tail = tail[:int(decimalPoints)]
	}
	for i := len(tail); i < int(decimalPoints); i++ {
		tail += "0"
	}
	result, ok := new(big.Int).SetString(head+tail, 10)
	if !ok {
		return nil, errInvalidNum(content)
	}

	return result, nil
}

func StringToBigInt(content string) (*big.Int, error) {
	value, err := decimal.NewFromString(content)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if value.Exponent() != 0 {
		return nil, errors.Errorf("can not covert %s to big.Int", content)
	}
	return value.BigInt(), nil
}

func HexStringToBigInt(content string) (*big.Int, error) {
	content = strings.ToLower(content)
	ok, _ := regexp.MatchString(`^(0x)?([0-9]|[a-f])+$`, content)
	if !ok {
		return nil, errInvalidNum(content)
	}
	content = strings.TrimPrefix(content, "0x")
	b, ok := new(big.Int).SetString(content, 16)
	if !ok {
		return nil, errInvalidNum(content)
	}
	return b, nil
}

func StringToUint64(val string) (uint64, error) {
	value, err := strconv.ParseUint(val, 0, 64)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	return value, nil
}

func MaxNum[T int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64](a, b T) T {
	if a < b {
		return b
	}
	return a
}
