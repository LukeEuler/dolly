package common

import (
	"database/sql/driver"
	"fmt"
	"math/big"
)

// BigInt 包装big.Int以实现sql.Scanner接口，可直接接收数据库中的Decimal
type BigInt struct {
	*big.Int
}

func NewBigInt(x int64) BigInt {
	return BigInt{
		big.NewInt(x),
	}
}

func WrapMathBig(x *big.Int) BigInt {
	return BigInt{
		x,
	}
}

func (b *BigInt) Add(bi BigInt) BigInt {
	v := new(big.Int).Set(b.Int)
	v = v.Add(v, bi.Origin())
	return WrapMathBig(v)
}

func (b *BigInt) Neg() BigInt {
	v := new(big.Int).Set(b.Int)
	v = v.Neg(v)
	return WrapMathBig(v)
}

func (b *BigInt) Origin() *big.Int {
	return b.Int
}

func (b *BigInt) Value() (driver.Value, error) {
	if b != nil {
		return (b).String(), nil
	}
	return nil, nil
}

func (b *BigInt) String() string {
	if b.Int == nil {
		return "0"
	}
	return b.Int.String()
}

// Scan assigns a value from a database driver.
//
// The src value will be of one of the following types:
//
//	int64
//	float64
//	bool
//	[]byte
//	string
//	time.Time
//	nil - for NULL values
//
// An error should be returned if the value cannot be stored
// without loss of information.
//
// Reference types such as []byte are only valid until the next call to Scan
// and should not be retained. Their underlying memory is owned by the driver.
// If retention is necessary, copy their values before the next call to Scan.
func (b *BigInt) Scan(value interface{}) error {
	b.Int = new(big.Int)
	if value == nil {
		return nil
	}
	switch t := value.(type) {
	case int64:
		b.SetInt64(t)
	case []byte:
		b.SetString(string(t), 10)
	case string:
		b.SetString(t, 10)
	default:
		return fmt.Errorf("could not scan type %T into BigInt ", t)
	}
	return nil
}
