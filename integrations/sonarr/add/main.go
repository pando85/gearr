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
	"golift.io/starr/sonarr"
)

type SeriesList []*sonarr.Series

func (s SeriesList) Len() int           { return len(s) }
func (s SeriesList) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s SeriesList) Less(i, j int) bool { return getSize(s[i]) > getSize(s[j]) }

func getSize(serie *sonarr.Series) int64 {
	if serie.Statistics != nil {
		return serie.Statistics.SizeOnDisk
	}
	return 0
}

func isNotX265OrH265(videoCodec string) bool {
	c := strings.ToLower(videoCodec)
	return (c != "x265" && c != "h265" && c != "hevc")
}

func isNotX265OrH265Serie(serie *sonarr.Series, s *sonarr.Sonarr) (bool, error) {
	episodeFiles, err := s.GetSeriesEpisodeFiles(serie.ID)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Cannot fetch episodes from serie: %s", serie.Title))
	}
	for _, e := range episodeFiles {
		if e.MediaInfo != nil {
			return isNotX265OrH265(e.MediaInfo.VideoCodec), nil
		}

	}
	return false, nil
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
		fmt.Println("Episode successfully queued.")
	case len(response.Failed) > 0:
		fmt.Println("Episode was not queued.")
	default:
		return errors.New("Episode was neither queued nor failed.")
	}

	return nil
}

func AddEpisodeToGearrQueue(path string, url string, token string) error {
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

	fmt.Println("Adding episode to gearr queue")

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
	apiKey := pflag.StringP("api-key", "k", "", "Sonarr API key")
	sonarrURL := pflag.StringP("url", "u", "", "Sonarr server URL")
	numSeries := pflag.Int("series", 10, "Number of series to retrieve")
	numEpisodes := pflag.Int("episodes", 1000, "Number of episodes to retrieve")
	gearrURL := pflag.String("gearr-url", "", "Gearr server URL")
	gearrToken := pflag.String("gearr-token", "", "Gearr web server token")
	dryRun := pflag.Bool("dry-run", false, "Dry run mode doesn't add episodes to gearr queue")

	pflag.Parse()

	if *apiKey == "" || *sonarrURL == "" || *gearrURL == "" || *gearrToken == "" {
		fmt.Println("Both API key and Sonarr URL are required.")
		pflag.PrintDefaults()
		return
	}

	gearrPostURL := fmt.Sprintf("%s/api/v1/job/", *gearrURL)

	c := starr.New(*apiKey, *sonarrURL, 0)
	s := sonarr.New(c)

	series, err := s.GetAllSeries()
	if err != nil {
		panic(err)
	}

	var filteredSeries SeriesList
	for _, serie := range series {
		isNotX265, err := isNotX265OrH265Serie(serie, s)
		if err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		if isNotX265 {
			filteredSeries = append(filteredSeries, serie)
		}
	}

	sort.Sort(filteredSeries)

	fmt.Printf("Number of filtered series: %d\n", len(filteredSeries))

	for i, serie := range filteredSeries {
		if i >= *numSeries {
			break
		}

		fmt.Printf("Title: %s\n", serie.Title)
		fmt.Printf("File Path: %s\n", serie.Path)

		episodeFiles, err := s.GetSeriesEpisodeFiles(serie.ID)
		if err != nil {
			fmt.Printf("Cannot fetch episodes from serie: %s", serie.Title)
			os.Exit(1)
		}

		var filteredEpisodes []*sonarr.EpisodeFile
		for _, e := range episodeFiles {
			if e.MediaInfo != nil && isNotX265OrH265(e.MediaInfo.VideoCodec) {
				filteredEpisodes = append(filteredEpisodes, e)
			}
		}

		for i, e := range filteredEpisodes {
			if i >= *numEpisodes {
				break
			}
			fmt.Printf("Codec: %s\n", e.MediaInfo.VideoCodec)

			fmt.Printf("Size: %s\n", HumanReadableSize(e.Size))
			fmt.Printf("Full Path: %s\n\n", e.Path)

			if !*dryRun {
				err := AddEpisodeToGearrQueue(e.Path, gearrPostURL, *gearrToken)
				if err != nil {
					fmt.Println("error:", err)
					os.Exit(1)
				}
			}
		}
	}
}
