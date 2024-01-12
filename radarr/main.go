package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"

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
	return m.MovieFile != nil && m.MovieFile.MediaInfo != nil &&
		(m.MovieFile.MediaInfo.VideoCodec != "x265" && m.MovieFile.MediaInfo.VideoCodec != "h265")
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

func AddMovieToTranscoderQueue(path string, url string) error {
	payload := map[string]string{
		"SourcePath": path,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := http.Client{}

	fmt.Println("Adding movie to transcoder queue")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		// Return an error with a detailed message
		return errors.New(fmt.Sprintf("Request failed with status %s. Response Body: %s", resp.Status, body))
	}

	return nil
}

func main() {
	apiKey := pflag.StringP("api-key", "k", "", "Radarr API key")
	radarrURL := pflag.StringP("url", "u", "", "Radarr server URL")
	numMovies := pflag.Int("movies", 10, "Number of movies to retrieve")
	transcoderURL := pflag.String("transcoder-url", "", "Transcoder server URL")
	transcoderToken := pflag.String("transcoder-token", "", "Transcoder web server token")
	dryRun := pflag.Bool("dry-run", false, "Dry run mode doesn't add movies to transcoder queue")

	pflag.Parse()

	if *apiKey == "" || *radarrURL == "" || *transcoderURL == "" || *transcoderToken == "" {
		fmt.Println("Both API key and Radarr URL are required.")
		pflag.PrintDefaults()
		return
	}

	transcoderPostURL := fmt.Sprintf("%s/api/v1/job/?token=%s", *transcoderURL, *transcoderToken)

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
				err := AddMovieToTranscoderQueue(m.MovieFile.Path, transcoderPostURL)
				if err != nil {
					fmt.Println("Error:", err)
					os.Exit(1)
				}
			}
		}
	}
}
