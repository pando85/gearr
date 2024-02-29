package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gearr/integrations/common"

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

func isNotX265OrH265Serie(serie *sonarr.Series, s *sonarr.Sonarr) bool {
	episodeFiles, err := s.GetSeriesEpisodeFiles(serie.ID)
	if err != nil {
		fmt.Printf("cannot fetch episodes from serie: %s", serie.Title)
		os.Exit(1)
	}
	for _, e := range episodeFiles {
		if e.MediaInfo != nil && isNotX265OrH265(e.MediaInfo.VideoCodec) {
			return true
		}
	}
	return false
}

func isNotX265OrH265(videoCodec string) bool {
	c := strings.ToLower(videoCodec)
	return (c != "x265" && c != "h265" && c != "hevc")
}

func main() {
	apiKey := pflag.StringP("api-key", "k", "", "Sonarr API key")
	sonarrURL := pflag.StringP("url", "u", "", "Sonarr server URL")
	numSeries := pflag.Int("series", 10, "Number of series to retrieve")
	numEpisodes := pflag.Int("episodes", 1000, "Number of episodes to retrieve")
	maxSize := pflag.Int64("max-size", 3000000000, "Number of episodes to retrieve")
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
		if isNotX265OrH265Serie(serie, s) {
			filteredSeries = append(filteredSeries, serie)
		}
	}

	sort.Sort(filteredSeries)

	fmt.Printf("number of filtered series: %d\n", len(filteredSeries))

	for i, serie := range filteredSeries {
		if i >= *numSeries {
			break
		}

		fmt.Printf("Title: %s\n", serie.Title)
		fmt.Printf("File Path: %s\n", serie.Path)

		episodeFiles, err := s.GetSeriesEpisodeFiles(serie.ID)
		if err != nil {
			fmt.Printf("cannot fetch episodes from serie: %s", serie.Title)
			os.Exit(1)
		}

		var filteredEpisodes []*sonarr.EpisodeFile
		for _, e := range episodeFiles {
			if e.MediaInfo == nil || isNotX265OrH265(e.MediaInfo.VideoCodec) || e.Size > *maxSize {
				filteredEpisodes = append(filteredEpisodes, e)
			}
		}

		fmt.Printf("number of episodes transcodeables %d from %d\n", len(filteredEpisodes), len(episodeFiles))

		for i, e := range filteredEpisodes {
			if i >= *numEpisodes {
				fmt.Print("max episodes added.")
				break
			}
			fmt.Printf("codec: %s\n", e.MediaInfo.VideoCodec)

			fmt.Printf("size: %s\n", common.HumanReadableSize(e.Size))
			fmt.Printf("full Path: %s\n\n", e.Path)

			if !*dryRun {
				err := common.AddToGearrQueue(e.Path, gearrPostURL, *gearrToken, "episode")
				if err != nil {
					fmt.Println("error:", err)
					os.Exit(1)
				}
			}
		}
	}
}
