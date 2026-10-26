package settings

import "encoding/json"

// Column is v as the json column a record's Shell attribute values are stored in; no values is "{}".
func (v AttributeValues) Column() ([]byte, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(v)
}

func AttributeValuesOf(column []byte) (AttributeValues, error) {
	values := AttributeValues{}
	err := json.Unmarshal(column, &values)
	return values, err
}
