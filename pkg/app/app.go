package app

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"webgears.org/immich-sync/pkg/config"
	"webgears.org/immich-sync/pkg/googlephotos"
	"webgears.org/immich-sync/pkg/immich"
)

type App struct {
	Cfg    *config.Config
	Client *immich.Client
	Logger *slog.Logger
}

func New(cfg *config.Config) (*App, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	client := immich.NewClient(cfg.ApiURL, cfg.ApiKey)
	return &App{
		Cfg:    cfg,
		Client: client,
		Logger: logger,
	}, nil
}

func (a *App) Run() {
	a.Logger.Info("Starting Immich Sync")

	id, name, err := a.Client.GetUser()
	if err != nil {
		a.Logger.Error("Failed to connect to Immich", "error", err)
		os.Exit(1)
	}
	a.Logger.Info("Connected to Immich", "user_id", id, "name", name)

	if len(a.Cfg.GooglePhotos) == 0 {
		a.Logger.Warn("No albums configured")
		return
	}

	forever := make(chan struct{})

	for _, ac := range a.Cfg.GooglePhotos {
		go a.syncLoop(ac)
	}

	<-forever
}

func (a *App) syncLoop(ac config.GooglePhotosConfig) {
	interval, err := time.ParseDuration(ac.SyncInterval)
	if err != nil || interval == 0 {
		interval = 24 * time.Hour
	}

	a.Logger.Info("Scheduled sync", "album", ac.URL, "interval", interval.String())

	// Run immediately
	a.runGPhotoSync(ac)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		<-ticker.C
		a.runGPhotoSync(ac)
	}
}

func (a *App) runGPhotoSync(ac config.GooglePhotosConfig) {
	logger := a.Logger.With("album_url", ac.URL)
	logger.Info("Syncing Google Photos Album")

	album, err := googlephotos.ScrapeAlbum(ac.URL)
	if err != nil {
		logger.Error("Error scraping album", "error", err)
		return
	}

	albumTitle := album.Title
	if ac.AlbumName != "" {
		albumTitle = ac.AlbumName
	}
	logger.Info("Found photos in album", "count", len(album.Photos), "title", albumTitle)

	// Ensure Immich Album exists
	var albumId string

	if ac.ImmichAlbumID != "" {
		albumId = ac.ImmichAlbumID
	} else {
		// Find or Create by Name
		albums, _ := a.Client.GetAlbums()
		for _, a := range albums {
			if a.AlbumName == albumTitle {
				albumId = a.Id
				break
			}
		}
		if albumId == "" {
			logger.Info("Creating Immich album", "title", albumTitle)
			newAlbum, err := a.Client.CreateAlbum(albumTitle)
			if err == nil {
				albumId = newAlbum.Id
			} else {
				logger.Error("Error creating album", "error", err)
			}
		}
	}

	var newAssetIds []string

	for _, p := range album.Photos {
		// Create a deterministic filename
		safeId := strings.ReplaceAll(p.ID, "/", "_")
		safeId = strings.ReplaceAll(safeId, ":", "_")
		fakeFilename := fmt.Sprintf("gp_%s.jpg", safeId)

		// Search in Immich
		exists, _ := a.Client.SearchAssets(fakeFilename)
		if len(exists) > 0 {
			// Found it.
			newAssetIds = append(newAssetIds, exists[0].Id)
			continue
		}

		logger.Info("Downloading new photo", "id", safeId)
		
		// Use Streaming Download & Upload
		r, size, err := googlephotos.DownloadPhotoStream(p.URL)
		if err != nil {
			logger.Error("Error downloading photo", "id", safeId, "error", err)
			continue
		}

		// Upload with metadata
        // Note: size is int64, which is correct for UploadAssetStream
		uploadedId, err := a.Client.UploadAssetStream(r, fakeFilename, size, p.TakenAt)
		r.Close() // Close response body
		
		if err != nil {
			logger.Error("Error uploading photo", "filename", fakeFilename, "error", err)
			continue
		}

		if uploadedId != "" {
			logger.Info("Uploaded photo", "filename", fakeFilename, "id", uploadedId)
			newAssetIds = append(newAssetIds, uploadedId)
		}
	}

	if albumId != "" && len(newAssetIds) > 0 {
		logger.Info("Adding assets to album", "count", len(newAssetIds), "album", albumTitle)
		err := a.Client.AddAssetsToAlbum(albumId, newAssetIds)
		if err != nil {
			logger.Error("Error adding assets to album", "error", err)
		}
	}
	logger.Info("Sync finished", "title", albumTitle)
}
