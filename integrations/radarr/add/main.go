package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/spf13/pflag"
	"golift.io/starr"
	"golift.io/starr/radarr"
)

type MovieBySize []*radarr.Movie

func (s MovieBySize) Len() int           { return len(s) }
func (s MovieBySize) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s MovieBySize) Less(i, j int) bool { return getSize(s[i]) > getSize(s[j]) }

func getSize(m *radarr.Movie) int64 {
	if m.MovieFile != nil {
		return m.MovieFile.Size
	}
	return 0
}

func isNotX265OrH265(m *radarr.Movie) bool {
	videoCodec := strings.ToLower(m.MovieFile.MediaInfo.VideoCodec)
	return m.MovieFile != nil && m.MovieFile.MediaInfo != nil &&
		(videoCodec != "x265" && videoCodec != "h265" && videoCodec != "hevc")
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
		fmt.Println("Movie successfully queued.")
	case len(response.Failed) > 0:
		fmt.Println("Movie was not queued.")
	default:
		return errors.New("Movie was neither queued nor failed.")
	}

	return nil
}

func AddMovieToGearrQueue(path string, url string, token string) error {
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

	fmt.Println("Adding movie to gearr queue")

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
		fmt.Fprintf(os.Stderr, "Failed with status %s and message: %s", resp.Status, body)
	}

	fmt.Println()
	fmt.Println()
	return nil
}

func main() {
	apiKey := pflag.StringP("api-key", "k", "", "Radarr API key")
	radarrURL := pflag.StringP("url", "u", "", "Radarr server URL")
	numMovies := pflag.Int("movies", 10, "Number of movies to retrieve")
	gearrURL := pflag.String("gearr-url", "", "Gearr server URL")
	gearrToken := pflag.String("gearr-token", "", "Gearr web server token")
	dryRun := pflag.Bool("dry-run", false, "Dry run mode doesn't add movies to gearr queue")

	pflag.Parse()

	if *apiKey == "" || *radarrURL == "" || *gearrURL == "" || *gearrToken == "" {
		fmt.Println("Both API key and Radarr URL are required.")
		pflag.PrintDefaults()
		return
	}

	gearrPostURL := fmt.Sprintf("%s/api/v1/job/", *gearrURL)

	c := starr.New(*apiKey, *radarrURL, 0)
	r := radarr.New(c)

	movies, err := r.GetMovie(0)
	if err != nil {
		panic(err)
	}

	var filteredMovies MovieBySize
	for _, m := range movies {
		if isNotX265OrH265(m) {
			filteredMovies = append(filteredMovies, m)
		}
	}

	sort.Sort(filteredMovies)

	fmt.Printf("Number of filtered movies: %d\n", len(filteredMovies))

	for i, m := range filteredMovies {
		if i >= *numMovies {
			break
		}

		fmt.Printf("Title: %s\n", m.Title)
		fmt.Printf("File Path: %s\n", m.Path)

		if m.MovieFile != nil {
			fmt.Printf("Codec: %s\n", m.MovieFile.MediaInfo.VideoCodec)

			fmt.Printf("Size: %s\n", HumanReadableSize(getSize(m)))
			fmt.Printf("Full Path: %s\n\n", m.MovieFile.Path)

			if !*dryRun {
				err := AddMovieToGearrQueue(m.MovieFile.Path, gearrPostURL, *gearrToken)
				if err != nil {
					fmt.Println("error:", err)
					os.Exit(1)
				}
			}
		}
	}
}
