package main

import (
	"errors"
	"time"

	"github.com/nstratos/go-myanimelist/mal"
	"github.com/rl404/verniy"
)

type MangaStatus string

const (
	MangaStatusReading    MangaStatus = "reading"
	MangaStatusCompleted  MangaStatus = "completed"
	MangaStatusOnHold     MangaStatus = "on_hold"
	MangaStatusDropped    MangaStatus = "dropped"
	MangaStatusPlanToRead MangaStatus = "plan_to_read"
	MangaStatusUnknown    MangaStatus = "unknown"
)

type Manga struct {
	IDAnilist       int
	IDMal           int
	Progress        int
	ProgressVolumes int
	Score           float64
	Status          MangaStatus
	TitleEN         string
	TitleJP         string
	TitleRomaji     string
	Chapters        int
	Volumes         int
	StartedAt       *time.Time
	FinishedAt      *time.Time
}

func newMangaFromMediaListEntry(mediaList verniy.MediaList) (Manga, error) {
	if mediaList.Media == nil {
		return Manga{}, errors.New("media is nil")
	}

	if mediaList.Status == nil {
		return Manga{}, errors.New("status is nil")
	}

	if mediaList.Media.Title == nil {
		return Manga{}, errors.New("title is nil")
	}

	var score float64
	if mediaList.Score != nil {
		score = *mediaList.Score
	}

	var progress int
	if mediaList.Progress != nil {
		progress = *mediaList.Progress
	}

	var progressVolumes int
	if mediaList.ProgressVolumes != nil {
		progressVolumes = *mediaList.ProgressVolumes
	}

	var titleEN string
	if mediaList.Media.Title.English != nil {
		titleEN = *mediaList.Media.Title.English
	}

	var titleJP string
	if mediaList.Media.Title.Native != nil {
		titleJP = *mediaList.Media.Title.Native
	}

	var idMal int
	if mediaList.Media.IDMAL != nil {
		idMal = *mediaList.Media.IDMAL
	}

	var romajiTitle string
	if mediaList.Media.Title.Romaji != nil {
		romajiTitle = *mediaList.Media.Title.Romaji
	}

	var chapters int
	if mediaList.Media.Chapters != nil {
		chapters = *mediaList.Media.Chapters
	}

	var volumes int
	if mediaList.Media.Volumes != nil {
		volumes = *mediaList.Media.Volumes
	}

	startedAt := convertFuzzyDateToTimeOrNow(mediaList.StartedAt)
	finishedAt := convertFuzzyDateToTimeOrNow(mediaList.CompletedAt)

	return Manga{
		IDAnilist:       mediaList.Media.ID,
		IDMal:           idMal,
		Progress:        progress,
		ProgressVolumes: progressVolumes,
		Score:           score,
		Status:          mapAnilistMangaStatustToStatus(*mediaList.Status),
		TitleEN:         titleEN,
		TitleJP:         titleJP,
		TitleRomaji:     romajiTitle,
		Chapters:        chapters,
		Volumes:         volumes,
		StartedAt:       startedAt,
		FinishedAt:      finishedAt,
	}, nil
}

func newMangaFromMalUserManga(manga mal.Manga) (Manga, error) {
	if manga.ID == 0 {
		return Manga{}, errors.New("ID is nil")
	}

	startedAt := parseDateOrNow(manga.MyListStatus.StartDate)
	finishedAt := parseDateOrNow(manga.MyListStatus.FinishDate)

	titleEN := manga.Title
	if manga.AlternativeTitles.En != "" {
		titleEN = manga.AlternativeTitles.En
	}

	titleJP := manga.Title
	if manga.AlternativeTitles.Ja != "" {
		titleJP = manga.AlternativeTitles.Ja
	}

	return Manga{
		IDAnilist:       -1,
		IDMal:           manga.ID,
		Progress:        manga.MyListStatus.NumChaptersRead,
		ProgressVolumes: manga.MyListStatus.NumVolumesRead,
		Score:           float64(manga.MyListStatus.Score),
		Status:          mapMalMangaStatusToStatus(manga.MyListStatus.Status),
		TitleEN:         titleEN,
		TitleJP:         titleJP,
		TitleRomaji:     "",
		Chapters:        manga.NumChapters,
		Volumes:         manga.NumVolumes,
		StartedAt:       startedAt,
		FinishedAt:      finishedAt,
	}, nil
}

func mapMalMangaStatusToStatus(s mal.MangaStatus) MangaStatus {
	switch s {
	case mal.MangaStatusReading:
		return MangaStatusReading
	case mal.MangaStatusCompleted:
		return MangaStatusCompleted
	case mal.MangaStatusOnHold:
		return MangaStatusOnHold
	case mal.MangaStatusDropped:
		return MangaStatusDropped
	case mal.MangaStatusPlanToRead:
		return MangaStatusPlanToRead
	default:
		return MangaStatusUnknown
	}
}

func mapAnilistMangaStatustToStatus(s verniy.MediaListStatus) MangaStatus {
	switch s {
	case verniy.MediaListStatusCurrent:
		return MangaStatusReading
	case verniy.MediaListStatusCompleted:
		return MangaStatusCompleted
	case verniy.MediaListStatusPaused:
		return MangaStatusOnHold
	case verniy.MediaListStatusDropped:
		return MangaStatusDropped
	case verniy.MediaListStatusPlanning:
		return MangaStatusPlanToRead
	case verniy.MediaListStatusRepeating:
		return MangaStatusReading // TODO: handle repeating correctly
	default:
		return MangaStatusUnknown
	}
}
