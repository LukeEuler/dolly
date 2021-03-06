package common

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
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
