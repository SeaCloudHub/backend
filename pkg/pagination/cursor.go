package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func DecodeCursor[T any](cursorStr string) (T, error) {
	var cursorObj T

	if len(cursorStr) == 0 {
		return cursorObj, nil
	}

	data, err := base64.URLEncoding.DecodeString(cursorStr)
	if err != nil {
		return cursorObj, fmt.Errorf("base64 decode: %w", err)
	}

	if err := json.Unmarshal(data, &cursorObj); err != nil {
		return cursorObj, fmt.Errorf("json unmarshal: %w", err)
	}

	return cursorObj, nil
}

func EncodeCursor[T any](cursor T) string {
	data, _ := json.Marshal(cursor)

	return base64.URLEncoding.EncodeToString(data)
}
