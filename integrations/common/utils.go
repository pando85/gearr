package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type ScheduledItem struct {
	SourcePath      string      `json:"source_path"`
	DestinationPath string      `json:"destination_path"`
	ID              string      `json:"id"`
	Events          interface{} `json:"events"`
}

type FailedItem struct {
	SourcePath      string `json:"source_path"`
	DestinationPath string `json:"destination_path"`
	ForceCompleted  bool   `json:"forceCompleted"`
	ForceFailed     bool   `json:"forceFailed"`
	ForceExecuting  bool   `json:"forceExecuting"`
	ForceQueued     bool   `json:"forceQueued"`
	Error           string `json:"error"`
}

type Response struct {
	Scheduled []ScheduledItem `json:"scheduled"`
	Failed    []FailedItem    `json:"failed"`
	Skipped   interface{}     `json:"skipped"`
}

func PrintGearrResponse(jsonStr []byte) error {
	var response Response
	if err := json.Unmarshal(jsonStr, &response); err != nil {
		return err
	}

	switch {
	case len(response.Scheduled) > 0:
		fmt.Println("item successfully queued.")
	case len(response.Failed) > 0:
		fmt.Println("item was not queued.")
	default:
		return fmt.Errorf("item was neither queued nor failed")
	}

	return nil
}

func AddToGearrQueue(path string, url string, token string, itemType string) error {
	payload := map[string]string{
		"source_path": path,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return err
	}

	authHeader := fmt.Sprintf("Bearer %s", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	client := http.Client{}

	fmt.Printf("adding %s %s to Gearr queue\n", itemType, path)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = PrintGearrResponse(body)
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "failed with status %s and message: %s", resp.Status, body)
	}

	fmt.Println()
	fmt.Println()
	return nil
}

func IsNotCodec(codec string, allowedCodecs []string) bool {
	lowerCodec := strings.ToLower(codec)
	for _, allowed := range allowedCodecs {
		if lowerCodec == allowed {
			return false
		}
	}
	return true
}

func HumanReadableSize(size int64) string {
	if size == 0 {
		return "N/A"
	}

	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
