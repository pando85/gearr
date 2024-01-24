package main

import (
	"fmt"
	"regexp"

	"github.com/spf13/pflag"
	"golift.io/starr"
	"golift.io/starr/radarr"
)

func isMovieInMovieArray(movie *radarr.Movie, mArray []radarr.Movie) bool {
	for _, m := range mArray {
		if m.Path == movie.Path {
			return true
		}
	}
	return false
}

func main() {
	apiKey := pflag.StringP("api-key", "k", "", "Radarr API key")
	radarrURL := pflag.StringP("url", "u", "", "Radarr server URL")
	dryRun := pflag.Bool("dry-run", false, "Dry run mode doesn't add movies to transcoder queue")

	pflag.Parse()
	regexPattern := `(.*)/[^\/]+_encoded\.mkv`

	// Get the input string from command line arguments
	if len(pflag.Args()) == 0 {
		fmt.Println("Usage: go run main.go <input_string>")
		return
	}
	inputString := pflag.Args()[0]

	regex := regexp.MustCompile(regexPattern)
	matches := regex.FindAllStringSubmatch(inputString, -1)

	moviesToUpdate := []radarr.Movie{}
	for _, match := range matches {
		movie := radarr.Movie{
			Path: match[1],
		}
		moviesToUpdate = append(moviesToUpdate, movie)
	}

	if *apiKey == "" || *radarrURL == "" {
		fmt.Println("Both API key and Radarr URL are required.")
		pflag.PrintDefaults()
		return
	}

	c := starr.New(*apiKey, *radarrURL, 0)
	r := radarr.New(c)

	movies, err := r.GetMovie(0)
	if err != nil {
		panic(err)
	}

	moviesIDs := []int64{}
	for _, m := range movies {
		if isMovieInMovieArray(m, moviesToUpdate) {
			fmt.Printf("Updating movie %s (%d)\n", m.Title, m.Year)
			moviesIDs = append(moviesIDs, m.ID)
		}
	}

	if !*dryRun {
		refreshCommand := radarr.CommandRequest{
			Name:     "RefreshMovie",
			MovieIDs: moviesIDs,
		}

		r.SendCommand(&refreshCommand)
	}
}
