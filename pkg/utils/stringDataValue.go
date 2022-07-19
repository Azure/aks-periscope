package utils

import (
	"io"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
)

type StringDataValue struct {
	value string
}

func NewStringDataValue(value string) *StringDataValue {
	return &StringDataValue{
		value: value,
	}
}

func (v *StringDataValue) GetLength() int64 {
	return int64(len(v.value))
}

func (v *StringDataValue) GetReader() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(v.value)), nil
}

func ToDataValueMap(data map[string]string) map[string]interfaces.DataValue {
	result := make(map[string]interfaces.DataValue, len(data))
	for key, value := range data {
		result[key] = NewStringDataValue(value)
	}

	return result
}
