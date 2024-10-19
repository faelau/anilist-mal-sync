package main

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/nstratos/go-myanimelist/mal"
	"github.com/rl404/verniy"
)

var errStatusUnknown = errors.New("status unknown")

var betweenBraketsRegexp = regexp.MustCompile(`\(.*\)`)

type Status string

const (
	StatusWatching    Status = "watching"
	StatusCompleted   Status = "completed"
	StatusOnHold      Status = "on_hold"
	StatusDropped     Status = "dropped"
	StatusPlanToWatch Status = "plan_to_watch"
	StatusUnknown     Status = "unknown"
)

func (s Status) GetMalStatus() (mal.AnimeStatus, error) {
	switch s {
	case StatusWatching:
		return mal.AnimeStatusWatching, nil
	case StatusCompleted:
		return mal.AnimeStatusCompleted, nil
	case StatusOnHold:
		return mal.AnimeStatusOnHold, nil
	case StatusDropped:
		return mal.AnimeStatusDropped, nil
	case StatusPlanToWatch:
		return mal.AnimeStatusPlanToWatch, nil
	default:
		return "", errStatusUnknown
	}
}

type Anime struct {
	NumEpisodes int
	IDAnilist   int
	IDMal       int
	Progress    int
	Score       float64
	SeasonYear  int
	Status      Status
	TitleEN     string
	TitleJP     string
	TitleRomaji string
	StartedAt   *time.Time
	FinishedAt  *time.Time
}

func (a Anime) GetTitle() string {
	if a.TitleEN != "" {
		return a.TitleEN
	}
	if a.TitleJP != "" {
		return a.TitleJP
	}
	return a.TitleRomaji
}

func (a Anime) IsSameAnime(b Anime) bool {
	if a.IDMal == b.IDMal {
		return true
	}

	eq := func(s1, s2 string) bool {
		if len(s1) < len(s2) {
			return strings.Contains(strings.ToLower(s2), strings.ToLower(s1))
		}
		return strings.Contains(strings.ToLower(s1), strings.ToLower(s2))
	}

	titlesEq := eq(a.TitleEN, b.TitleEN)
	if !titlesEq {
		titlesEq = eq(a.TitleJP, b.TitleJP)
	}

	if titlesEq {
		return true
	}

	f := func(s1, s2 string) bool {
		if len(s1) < len(s2) {
			s1, s2 = s2, s1
		}

		c := 0
		for i, r := range s1 {
			if r == rune(s2[i]) {
				c = i
			} else {
				break
			}
		}

		return float64(c)/float64(len(s1))*100 > 80
	}

	// JP
	aa := strings.ReplaceAll(a.TitleJP, " ", "")
	bb := strings.ReplaceAll(b.TitleJP, " ", "")

	if f(aa, bb) {
		return true
	}

	// EN
	aa = strings.ReplaceAll(a.TitleEN, " ", "")
	bb = strings.ReplaceAll(b.TitleEN, " ", "")

	if f(aa, bb) {
		return true
	}

	aa = betweenBraketsRegexp.ReplaceAllString(aa, "")
	bb = betweenBraketsRegexp.ReplaceAllString(bb, "")

	return f(aa, bb)
}

func (a Anime) IsSameProgress(b Anime) bool {
	if a.Status != b.Status {
		if debug {
			log.Printf("Status: %s != %s", a.Status, b.Status)
		}
		return false
	}
	if a.Score != b.Score {
		if debug {
			log.Printf("Score: %f != %f", a.Score, b.Score)
		}
		return false
	}
	progress := a.Progress == b.Progress
	if a.NumEpisodes == b.NumEpisodes {
		if debug {
			log.Printf("Equal number of episodes: %d == %d", a.NumEpisodes, b.NumEpisodes)
			log.Printf("Progress: %t", progress)
		}
		return progress
	}
	if a.NumEpisodes == 0 || b.NumEpisodes == 0 {
		if debug {
			log.Printf("One of the anime has 0 episodes: %d, %d", a.NumEpisodes, b.NumEpisodes)
			log.Printf("Progress: %t", progress)
		}
		return progress
	}
	if progress && (a.NumEpisodes-b.NumEpisodes != 0) {
		if debug {
			log.Printf("Both anime have 0 progress but different number of episodes: %d, %d", a.NumEpisodes, b.NumEpisodes)
		}
		return true
	}

	aa := (a.NumEpisodes - a.Progress)
	bb := (b.NumEpisodes - b.Progress)
	if debug {
		log.Printf("Number of episodes: %d, %d", a.NumEpisodes, b.NumEpisodes)
		log.Printf("Progress: %d, %d", a.Progress, b.Progress)
		log.Printf("Progress: %d == %d", aa, bb)
	}
	return aa == bb
}

func (a Anime) IsSameDates(b Anime) bool {
	if a.StartedAt == nil && b.StartedAt == nil {
		return true
	}
	if a.StartedAt == nil || b.StartedAt == nil {
		return false
	}
	if a.FinishedAt == nil && b.FinishedAt == nil {
		return true
	}
	if a.FinishedAt == nil || b.FinishedAt == nil {
		return false
	}
	return a.StartedAt.Equal(*b.StartedAt) && a.FinishedAt.Equal(*b.FinishedAt)
}

func (a Anime) DiffString(b Anime) string {
	sb := strings.Builder{}
	sb.WriteString("Diff{")
	if a.Status != b.Status {
		sb.WriteString(fmt.Sprintf("Status: %s -> %s, ", a.Status, b.Status))
	}
	if a.Score != b.Score {
		sb.WriteString(fmt.Sprintf("Score: %f -> %f, ", a.Score, b.Score))
	}
	if a.Progress != b.Progress {
		sb.WriteString(fmt.Sprintf("Progress: %d -> %d, ", a.Progress, b.Progress))
	}
	if a.NumEpisodes != b.NumEpisodes {
		sb.WriteString(fmt.Sprintf("NumEpisodes: %d -> %d, ", a.NumEpisodes, b.NumEpisodes))
	}
	// if a.StartedAt != b.StartedAt {
	// 	sb.WriteString(fmt.Sprintf("StartedAt: %s -> %s, ", a.StartedAt, b.StartedAt))
	// }
	// if a.FinishedAt != b.FinishedAt {
	// 	sb.WriteString(fmt.Sprintf("FinishedAt: %s -> %s, ", a.FinishedAt, b.FinishedAt))
	// }
	sb.WriteString("}")
	return sb.String()
}

func (a Anime) String() string {
	sb := strings.Builder{}
	sb.WriteString("Anime{")
	sb.WriteString(fmt.Sprintf("IDAnilist: %d, ", a.IDAnilist))
	sb.WriteString(fmt.Sprintf("IDMal: %d, ", a.IDMal))
	sb.WriteString(fmt.Sprintf("TitleEN: %s, ", a.TitleEN))
	sb.WriteString(fmt.Sprintf("TitleJP: %s, ", a.TitleJP))
	sb.WriteString(fmt.Sprintf("MediaListStatus: %s, ", a.Status))
	sb.WriteString(fmt.Sprintf("Score: %f, ", a.Score))
	sb.WriteString(fmt.Sprintf("Progress: %d, ", a.Progress))
	sb.WriteString(fmt.Sprintf("EpisodeNumber: %d, ", a.NumEpisodes))
	sb.WriteString(fmt.Sprintf("SeasonYear: %d, ", a.SeasonYear))
	sb.WriteString(fmt.Sprintf("StartedAt: %s, ", a.StartedAt))
	sb.WriteString(fmt.Sprintf("FinishedAt: %s", a.FinishedAt))
	sb.WriteString("}")
	return sb.String()
}

func newAmimeFromMediaListEntry(mediaList verniy.MediaList) (Anime, error) {
	if mediaList.Media == nil {
		return Anime{}, errors.New("media is nil")
	}

	if mediaList.Status == nil {
		return Anime{}, errors.New("status is nil")
	}

	if mediaList.Media.Title == nil {
		return Anime{}, errors.New("title is nil")
	}

	var score float64
	if mediaList.Score != nil {
		score = *mediaList.Score
	}

	var progress int
	if mediaList.Progress != nil {
		progress = *mediaList.Progress
	}

	var titleEN string
	if mediaList.Media.Title.English != nil {
		titleEN = *mediaList.Media.Title.English
	}

	var titleJP string
	if mediaList.Media.Title.Native != nil {
		titleJP = *mediaList.Media.Title.Native
	}

	var episodeNumber int
	if mediaList.Media.Episodes != nil {
		episodeNumber = *mediaList.Media.Episodes
	}

	var year int
	if mediaList.Media.SeasonYear != nil {
		year = *mediaList.Media.SeasonYear
	}

	var idMal int
	if mediaList.Media.IDMAL != nil {
		idMal = *mediaList.Media.IDMAL
	}

	var romajiTitle string
	if mediaList.Media.Title.Romaji != nil {
		romajiTitle = *mediaList.Media.Title.Romaji
	}

	startedAt := convertFuzzyDateToTimeOrNow(mediaList.StartedAt)
	finishedAt := convertFuzzyDateToTimeOrNow(mediaList.CompletedAt)

	return Anime{
		NumEpisodes: episodeNumber,
		IDAnilist:   mediaList.Media.ID,
		IDMal:       idMal,
		Progress:    progress,
		Score:       score,
		SeasonYear:  year,
		Status:      mapVerniyStatusToStatus(*mediaList.Status),
		TitleEN:     titleEN,
		TitleJP:     titleJP,
		TitleRomaji: romajiTitle,
		StartedAt:   startedAt,
		FinishedAt:  finishedAt,
	}, nil
}

func newAnimeFromMalAnime(malAnime mal.Anime) (Anime, error) {
	if malAnime.ID == 0 {
		return Anime{}, errors.New("ID is nil")
	}

	startedAt := parseDateOrNow(malAnime.MyListStatus.StartDate)
	finishedAt := parseDateOrNow(malAnime.MyListStatus.FinishDate)

	titleEN := malAnime.Title
	if malAnime.AlternativeTitles.En != "" {
		titleEN = malAnime.AlternativeTitles.En
	}

	titleJP := malAnime.Title
	if malAnime.AlternativeTitles.Ja != "" {
		titleJP = malAnime.AlternativeTitles.Ja
	}

	return Anime{
		NumEpisodes: malAnime.NumEpisodes,
		IDAnilist:   -1,
		IDMal:       malAnime.ID,
		Progress:    malAnime.MyListStatus.NumEpisodesWatched,
		Score:       float64(malAnime.MyListStatus.Score),
		SeasonYear:  malAnime.StartSeason.Year,
		Status:      mapMalAnimeStatusToStatus(malAnime.MyListStatus.Status),
		TitleEN:     titleEN,
		TitleJP:     titleJP,
		StartedAt:   startedAt,
		FinishedAt:  finishedAt,
	}, nil
}

func mapVerniyStatusToStatus(s verniy.MediaListStatus) Status {
	switch s {
	case verniy.MediaListStatusCurrent:
		return StatusWatching
	case verniy.MediaListStatusCompleted:
		return StatusCompleted
	case verniy.MediaListStatusPaused:
		return StatusOnHold
	case verniy.MediaListStatusDropped:
		return StatusDropped
	case verniy.MediaListStatusPlanning:
		return StatusPlanToWatch
	default:
		return StatusUnknown
	}
}

func mapMalAnimeStatusToStatus(s mal.AnimeStatus) Status {
	switch s {
	case mal.AnimeStatusWatching:
		return StatusWatching
	case mal.AnimeStatusCompleted:
		return StatusCompleted
	case mal.AnimeStatusOnHold:
		return StatusOnHold
	case mal.AnimeStatusDropped:
		return StatusDropped
	case mal.AnimeStatusPlanToWatch:
		return StatusPlanToWatch
	default:
		return StatusUnknown
	}
}

func convertFuzzyDateToTimeOrNow(fd *verniy.FuzzyDate) *time.Time {
	if fd == nil || fd.Year == nil || fd.Month == nil || fd.Day == nil {
		return nil
	}
	d := time.Date(
		*fd.Year,
		time.Month(*fd.Month),
		*fd.Day,
		0, 0, 0, 0,
		time.UTC,
	)
	return &d
}

func parseDateOrNow(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}
	parsedTime, err := time.Parse(time.DateOnly, dateStr)
	if err != nil {
		return nil
	}
	parsedTime = parsedTime.UTC().Truncate(24 * time.Hour)
	return &parsedTime
}
