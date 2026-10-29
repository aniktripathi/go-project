package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MonthPoint is one month of total expenses sent to the optional ML service.
type MonthPoint struct {
	Month string  `json:"month"`
	Total float64 `json:"total"`
}

// FetchPrediction calls the optional Python service. Any failure is returned as an
// error so the caller can show "prediction unavailable" while the app keeps working.
func FetchPrediction(baseURL string, history []MonthPoint) (map[string]interface{}, error) {
	body, err := json.Marshal(map[string]interface{}{"history": history})
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post(baseURL+"/predict", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prediction service returned status %d", resp.StatusCode)
	}
	var out map[string]interface{}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
