package common

import (
	"github.com/pkg/errors"
)

// uint64 -> uint64 -> BigInt
type ChangeAmountMapTypeA struct {
	// first id => second id => amount
	Content     map[uint64]map[uint64]BigInt
	LastContent map[uint64]map[uint64]BigInt // 记录上一批余额信息
}

func NewChangeAmountMapTypeA() *ChangeAmountMapTypeA {
	return &ChangeAmountMapTypeA{Content: make(map[uint64]map[uint64]BigInt)}
}

// Change add changes
func (m *ChangeAmountMapTypeA) Change(firstID, secondID uint64, changeAmount BigInt) {
	if m.Content == nil {
		m.Content = make(map[uint64]map[uint64]BigInt)
	}
	if _, ok := m.Content[firstID]; !ok {
		m.Content[firstID] = make(map[uint64]BigInt)
	}
	if _, ok := m.Content[firstID][secondID]; !ok {
		m.Content[firstID][secondID] = NewBigInt(0)
	}
	amount := m.Content[firstID][secondID]
	m.Content[firstID][secondID] = amount.Add(changeAmount)
}

func (m *ChangeAmountMapTypeA) lastValueExist(firstID, secondID uint64) bool {
	_, ok := m.LastContent[firstID]
	if !ok {
		return false
	}
	_, ok = m.LastContent[firstID][secondID]
	return ok
}

func (m *ChangeAmountMapTypeA) initLastValue(firstID, secondID uint64, f func(uint64, uint64) (BigInt, error)) error {
	if m.lastValueExist(firstID, secondID) {
		return nil
	}
	value, err := f(firstID, secondID)
	if err != nil {
		return err
	}
	m.setLastValue(firstID, secondID, value)
	return nil
}

func (m *ChangeAmountMapTypeA) setLastValue(firstID, secondID uint64, amount BigInt) {
	if m.LastContent == nil {
		m.LastContent = make(map[uint64]map[uint64]BigInt)
	}
	if _, ok := m.LastContent[firstID]; !ok {
		m.LastContent[firstID] = make(map[uint64]BigInt)
	}
	m.LastContent[firstID][secondID] = amount
}

func (m *ChangeAmountMapTypeA) GetNowValue(firstID, secondID uint64) (BigInt, error) {
	var v = NewBigInt(0)
	_, ok := m.Content[firstID]
	if !ok {
		return v, errors.Errorf("nothing about first id %d", firstID)
	}
	_, ok = m.LastContent[firstID]
	if !ok {
		return v, errors.Errorf("nothing about first id %d", firstID)
	}

	v1, ok := m.Content[firstID][secondID]
	if !ok {
		return v, errors.Errorf("nothing about first id %d, second id %d", firstID, secondID)
	}
	v2, ok := m.LastContent[firstID][secondID]
	if !ok {
		return v, errors.Errorf("nothing about first id %d, second id %d", firstID, secondID)
	}

	v = v1.Add(v2)
	return v, nil
}

func (m *ChangeAmountMapTypeA) InitAndGetNowValue(firstID, secondID uint64, f func(uint64, uint64) (BigInt, error)) (BigInt, error) {
	err := m.initLastValue(firstID, secondID, f)
	if err != nil {
		return NewBigInt(0), err
	}
	return m.GetNowValue(firstID, secondID)
}
