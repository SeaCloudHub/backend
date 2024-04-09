package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type Cursor struct {
	Token     string
	Limit     int
	nextToken string
}

func NewCursor(token string, limit int) *Cursor {
	return &Cursor{
		Token: token,
		Limit: limit,
	}
}

func (c *Cursor) NextToken() string {
	return c.nextToken
}

func (c *Cursor) SetNextToken(token string) {
	c.nextToken = token
}

func DecodeToken[T any](tokenStr string) (T, error) {
	var tokenObj T

	if len(tokenStr) == 0 {
		return tokenObj, nil
	}

	data, err := base64.URLEncoding.DecodeString(tokenStr)
	if err != nil {
		return tokenObj, fmt.Errorf("base64 decode: %w", err)
	}

	if err := json.Unmarshal(data, &tokenObj); err != nil {
		return tokenObj, fmt.Errorf("json unmarshal: %w", err)
	}

	return tokenObj, nil
}

func EncodeToken[T any](token T) string {
	data, _ := json.Marshal(token)

	return base64.URLEncoding.EncodeToString(data)
}
