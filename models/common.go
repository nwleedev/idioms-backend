package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type TextArray []string

func (array *TextArray) Scan(src interface{}) error {
	switch casted := src.(type) {
	case []uint8:
		{
			temps := []string{}
			err := json.Unmarshal(casted, &temps)

			if err != nil {
				return err
			}
			for _, temp := range temps {
				*array = append(*array, temp)
			}
			break
		}
	case nil:
		{
			return nil
		}
	default:
		{
			return errors.New("invalid value to scan")
		}
	}
	return nil
}

func (array TextArray) Value() (driver.Value, error) {
	if array == nil {
		return nil, errors.New("array is nil")
	}

	buf, err := json.Marshal(array)
	if err != nil {
		return nil, err
	}
	return buf, nil
}
