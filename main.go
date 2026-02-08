package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"webgears.org/immich-sync/pkg/config"
	"webgears.org/immich-sync/pkg/googlephotos"
	"webgears.org/immich-sync/pkg/immich"
)

var ic *immich.Client

func main() {
	fmt.Println(">> Immich Sync Tool <<")
	
	cfg, err := config.ReadConfig("config.json")
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		if os.Getenv("IMMICH_API_KEY") == "" {
			fmt.Println("Please provide config.json or environment variables.")
			os.Exit(1)
		}
	}

	ic = immich.NewClient(cfg.ApiURL, cfg.ApiKey)
	id, name, err := ic.GetUser()
	if err != nil {
		log.Fatalf("Failed to connect to Immich: %v", err)
	}
	fmt.Printf("Connected to Immich as %s (%s)\n", name, id)

	if len(cfg.GooglePhotos) == 0 {
		fmt.Println("No Google Photos albums configured in config.json")
		return
	}

	fmt.Println("Starting Google Photos Sync Service...")
	startGPhotosSync(cfg.GooglePhotos)
	
	// Block main thread to keep background routines running
	select {}
}

func startGPhotosSync(albumConfigs []config.GooglePhotosConfig) {
	for _, ac := range albumConfigs {
		go syncLoop(ac)
	}
}

func syncLoop(ac config.GooglePhotosConfig) {
	interval, err := time.ParseDuration(ac.SyncInterval)
	if err != nil || interval == 0 {
		interval = 24 * time.Hour
	}

	fmt.Printf("Scheduled sync for album %s every %s\n", ac.URL, interval)

	// Run immediately
	runGPhotoSync(ac)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		<-ticker.C
		runGPhotoSync(ac)
	}
}

func runGPhotoSync(ac config.GooglePhotosConfig) {
	fmt.Printf("Syncing Google Photos Album: %s\n", ac.URL)
	album, err := googlephotos.ScrapeAlbum(ac.URL)
	if err != nil {
		fmt.Printf("Error scraping album at %s: %v\n", ac.URL, err)
		return
	}
	
	albumTitle := album.Title
	if ac.AlbumName != "" {
		albumTitle = ac.AlbumName // Override from config
	}
	
	fmt.Printf("Found %d photos in album '%s'\n", len(album.Photos), albumTitle)

	// Ensure Immich Album exists
	var albumId string
	
	if ac.ImmichAlbumID != "" {
		albumId = ac.ImmichAlbumID
	} else {
		// Find or Create by Name
		albums, _ := ic.GetAlbums()
		for _, a := range albums {
			if a.AlbumName == albumTitle {
				albumId = a.Id
				break
			}
		}
		if albumId == "" {
			fmt.Printf("Creating Immich album: %s\n", albumTitle)
			newAlbum, err := ic.CreateAlbum(albumTitle)
			if err == nil {
				albumId = newAlbum.Id
			} else {
				fmt.Printf("Error creating album: %v\n", err)
				// Continue without adding to album if creation fails? 
				// Better to stop maybe? But assets can still be uploaded.
			}
		}
	}

	// Process photos
	var newAssetIds []string
	
	tmpDir := os.TempDir()
	
	for _, p := range album.Photos {
		// Create a deterministic filename
		safeId := strings.ReplaceAll(p.ID, "/", "_")
		safeId = strings.ReplaceAll(safeId, ":", "_")
		// Use a prefix to avoid collision with other uploads?
		// User might want original filenames but we don't have them easily from scraped URL unless headers.
		// DownloadPhoto could return filename from header.
		// For now simple deterministic name:
		fakeFilename := fmt.Sprintf("gp_%s.jpg", safeId)

		// Search in Immich
		exists, _ := ic.SearchAssets(fakeFilename)
		if len(exists) > 0 {
			// Found it. 
			// We skip download & upload
			// But we might need to add to album if not already there.
			// Currently we always collect IDs to add to album. Immich API handles already-in-album gracefully.
			newAssetIds = append(newAssetIds, exists[0].Id)
			continue
		}

		// Download
		// fmt.Printf("Downloading photo %s...\n", safeId) // verbose
		data, err := googlephotos.DownloadPhoto(p.URL)
		if err != nil {
			fmt.Printf("Error downloading %s: %v\n", safeId, err)
			continue
		}

		tmpFile := filepath.Join(tmpDir, fakeFilename)
		err = os.WriteFile(tmpFile, data, 0644)
		if err != nil {
			continue
		}

		// Upload with metadata
		uploadedId, err := ic.UploadAsset(tmpFile, p.TakenAt)
		os.Remove(tmpFile) // clean up
		
		if err != nil {
			fmt.Printf("Error uploading %s: %v\n", fakeFilename, err)
			continue
		}
		
		if uploadedId != "" {
			fmt.Printf("Uploaded: %s\n", fakeFilename)
			newAssetIds = append(newAssetIds, uploadedId)
		}
	}

	if albumId != "" && len(newAssetIds) > 0 {
		fmt.Printf("Adding %d assets to album %s\n", len(newAssetIds), albumTitle)
		err := ic.AddAssetsToAlbum(albumId, newAssetIds)
		if err != nil {
			fmt.Printf("Error adding assets to album: %v\n", err)
		}
	}
	fmt.Printf("Sync finished for '%s'\n", albumTitle)
}
