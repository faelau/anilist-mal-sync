package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/nstratos/go-myanimelist/mal"
	"github.com/rl404/verniy"
)

type Statistics struct {
	UpdatedCount int
	SkippedCount int
	TotalCount   int
}

func (s *Statistics) Print() {
	fmt.Printf("Updated %d out of %d animes\n", s.UpdatedCount, s.TotalCount)
	fmt.Printf("Skipped %d animes\n", s.SkippedCount)
}

type App struct {
	config    Config
	mal       *MyAnimeListClient
	anilist   *AnilistClient
	forceSync bool
	dryRun    bool
	mangaSync bool
	stats     *Statistics
	ignore    map[string]struct{}
}

func NewApp(ctx context.Context, config Config, forceSync bool, dryRun bool, syncManga bool) (*App, error) {
	oauthMAL, err := NewMyAnimeListOAuth(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("error creating mal oauth: %w", err)
	}

	log.Println("Got MAL token")

	malClient, err := NewMyAnimeListClient(ctx, oauthMAL, config.MyAnimeList.Username)
	if err != nil {
		return nil, fmt.Errorf("error creating mal client: %w", err)
	}

	log.Println("MAL client created")

	oauthAnilist, err := NewAnilistOAuth(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("error creating anilist oauth: %w", err)
	}

	log.Println("Got Anilist token")

	anilistClient, err := NewAnilistClient(ctx, oauthAnilist, config.Anilist.Username)
	if err != nil {
		return nil, fmt.Errorf("error creating anilist client: %w", err)
	}

	log.Println("Anilist client created")

	return &App{
		config:    config,
		mal:       malClient,
		anilist:   anilistClient,
		forceSync: forceSync,
		dryRun:    dryRun,
		mangaSync: syncManga,
		stats:     &Statistics{},
		ignore: map[string]struct{}{ // in lowercase, TODO: move to config
			"scott pilgrim takes off":       {}, // this anime is not in MAL
			"bocchi the rock! recap part 2": {}, // this anime is not in MAL
		},
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	if a.mangaSync {
		log.Println("Syncing manga")

		return a.syncManga(ctx)
	}

	log.Println("Syncing anime")

	return a.syncAnime(ctx)
}

func (a *App) syncAnime(ctx context.Context) error {
	srcAnimeList, err := a.anilist.GetUserAnimeList(ctx)
	if err != nil {
		return fmt.Errorf("error getting user anime list: %w", err)
	}

	count := countAnimes(srcAnimeList)
	log.Printf("Got %d animes from AniList", count)

	tgtAnimeMap, err := a.getTargetAnimeMap(ctx)
	if err != nil {
		return err
	}

	a.processAnimeList(ctx, srcAnimeList, tgtAnimeMap)

	log.Printf("--------------------------------")
	a.stats.Print()

	return nil
}

func (a *App) syncManga(ctx context.Context) error {
	r, err := a.anilist.GetUserMangaList(ctx)
	if err != nil {
		return fmt.Errorf("error getting user manga list: %w", err)
	}

	if data, err := json.Marshal(r); err == nil {
		fmt.Println(string(data))
	}

	return nil
}

func (a *App) getTargetAnimeMap(ctx context.Context) (map[int]Anime, error) {
	if a.forceSync {
		log.Println("Forcing sync, skipping MAL fetch")
		return nil, nil
	}

	tgtAnimeList, err := a.mal.GetUserAnimeList(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting user anime list: %w", err)
	}

	log.Printf("Got %d animes from MAL", len(tgtAnimeList))

	tgtAnimeMap := make(map[int]Anime)
	for _, a := range tgtAnimeList {
		tgt, err := newAnimeFromMalAnime(a.Anime)
		if err != nil {
			return nil, fmt.Errorf("error creating anime: %w", err)
		}
		tgtAnimeMap[tgt.IDMal] = tgt
	}
	return tgtAnimeMap, nil
}

func (a *App) processAnimeList(ctx context.Context, srcAnimeList []verniy.MediaListGroup, tgtAnimeMap map[int]Anime) {
	for _, listEntry := range srcAnimeList {
		if listEntry.Status == nil {
			continue
		}

		log.Printf("--------------------------------")
		log.Printf("Processing for status: %s", *listEntry.Status)

		for _, entry := range listEntry.Entries {
			a.stats.TotalCount++

			src, err := newAmimeFromMediaListEntry(entry)
			if err != nil {
				log.Printf("error creating anime: %v", err)
				continue
			}

			if _, ok := a.ignore[strings.ToLower(src.GetTitle())]; ok {
				log.Printf("Ignoring anime: %s", src.GetTitle())
				continue
			}

			a.processSingleAnime(ctx, src, tgtAnimeMap)
		}
	}
}

func (a *App) processSingleAnime(ctx context.Context, src Anime, tgtAnimeMap map[int]Anime) {
	if !a.forceSync {
		tgt, err := a.getOrFetchTargetAnime(ctx, src, tgtAnimeMap)
		if err != nil {
			log.Printf("error processing target anime: %v", err)
			return
		}

		if src.IsSameProgress(tgt) /* && src.IsSameDates(tgt) */ {
			a.stats.SkippedCount++
			return
		}

		log.Print("--------------------------------")
		log.Printf("Title: %s", src.GetTitle())
		log.Printf("Progress is not same, need to update: %s", src.DiffString(tgt))
	}

	if a.dryRun {
		log.Printf("Dry run: Skipping update for anime %s", src.GetTitle())
		return
	}

	a.updateAnime(ctx, src)
}

func (a *App) getOrFetchTargetAnime(ctx context.Context, src Anime, tgtAnimeMap map[int]Anime) (Anime, error) {
	if tgt, ok := tgtAnimeMap[src.IDMal]; ok {
		return tgt, nil
	}

	f := func(a mal.Anime) (Anime, error) {
		tgt, err := newAnimeFromMalAnime(a)
		if err != nil {
			return Anime{}, fmt.Errorf("error creating anime: %w: %+v", err, a)
		}

		if !src.IsSameAnime(tgt) {
			log.Printf("Different animes: \nsrc: %+v\ntgt: %+v", src, tgt)
			return Anime{}, fmt.Errorf("animes are different: %+v, %+v", src, tgt)
		}
		return tgt, nil
	}

	malAnime, err := a.mal.GetAnimeByID(ctx, src.IDMal)
	switch {
	case err == nil:
		return f(*malAnime)
	case errors.Is(err, errEmptyMalID):
		malAnimeList, err := a.mal.GetAnimesByName(ctx, src.GetTitle())
		if err != nil {
			return Anime{}, fmt.Errorf("error getting mal anime: %s, %w", src.GetTitle(), err)
		}

		for _, a := range malAnimeList {
			tgt, err := f(a)
			if err == nil {
				return tgt, nil
			}

			log.Printf("error getting mal anime by name: %v", err)
			continue
		}

		return Anime{}, fmt.Errorf("error getting mal anime by name: %s: not found", src.GetTitle())
	default:
		return Anime{}, fmt.Errorf("error getting mal anime by id: %s: %w", src.GetTitle(), err)
	}
}

func (a *App) updateAnime(ctx context.Context, src Anime) {
	log.Printf("Updating anime from AniList to MAL")

	err := a.mal.UpdateAnime(ctx, src)
	if err != nil {
		if errors.Is(err, errEmptyMalID) {
			log.Printf("Not found in MAL")
			return
		}
		log.Printf("error updating anime: %v", err)
		return
	}

	log.Printf("Anime updated")
	a.stats.UpdatedCount++
}

func countAnimes(srcAnimeList []verniy.MediaListGroup) int {
	var count int
	for _, a := range srcAnimeList {
		count += len(a.Entries)
	}
	return count
}
