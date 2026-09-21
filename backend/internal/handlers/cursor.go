package handlers

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

type dateIDCursor struct {
	Date string
	ID   int64
}

func encodeDateIDCursor(date string, id int64) string {
	payload, _ := json.Marshal(map[string]any{"d": date, "i": id})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeDateIDCursor(raw string) (dateIDCursor, bool) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		padded := raw
		switch len(padded) % 4 {
		case 2:
			padded += "=="
		case 3:
			padded += "="
		}
		b, err = base64.URLEncoding.DecodeString(padded)
		if err != nil {
			return dateIDCursor{}, false
		}
	}
	var data struct {
		D string `json:"d"`
		I int64  `json:"i"`
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return dateIDCursor{}, false
	}
	date := strings.TrimSpace(data.D)
	if len(date) < 10 || data.I < 1 {
		return dateIDCursor{}, false
	}
	return dateIDCursor{Date: date[:10], ID: data.I}, true
}
