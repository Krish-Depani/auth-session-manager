package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func GetIPLocation(ipAddress string) string {
	if ipAddress == "127.0.0.1" || ipAddress == "::1" || strings.HasPrefix(ipAddress, "192.168.") {
		return "Local"
	}

	url := fmt.Sprintf("http://ip-api.com/json/%s", ipAddress)
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return "Unknown"
	}
	defer resp.Body.Close()

	var result struct {
		Country string `json:"country"`
		City    string `json:"city"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "Unknown"
	}

	if result.City != "" && result.Country != "" {
		return fmt.Sprintf("%s, %s", result.City, result.Country)
	}

	return "Unknown"
}
