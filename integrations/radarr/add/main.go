package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gearr/integrations/common"

	"github.com/spf13/pflag"
	"golift.io/starr"
	"golift.io/starr/radarr"
)

type MoviesList []*radarr.Movie

func (s MoviesList) Len() int           { return len(s) }
func (s MoviesList) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s MoviesList) Less(i, j int) bool { return getSize(s[i]) > getSize(s[j]) }

func getSize(m *radarr.Movie) int64 {
	if m.MovieFile != nil {
		return m.MovieFile.Size
	}
	return 0
}

func isNotX265OrH265(m *radarr.Movie) bool {
	videoCodec := strings.ToLower(m.MovieFile.MediaInfo.VideoCodec)
	return m.MovieFile != nil && m.MovieFile.MediaInfo != nil &&
		common.IsNotCodec(videoCodec, []string{"x265", "h265", "hevc"})
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

	var filteredMovies MoviesList
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

			fmt.Printf("Size: %s\n", common.HumanReadableSize(getSize(m)))
			fmt.Printf("Full Path: %s\n\n", m.MovieFile.Path)

			if !*dryRun {
				err := common.AddToGearrQueue(m.MovieFile.Path, gearrPostURL, *gearrToken, "movie")
				if err != nil {
					fmt.Println("error:", err)
					os.Exit(1)
				}
			}
		}
	}
}
