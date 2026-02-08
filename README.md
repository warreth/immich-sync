# Immich Sync

Sync photos from Google Photos Shared Albums to your Immich instance.

## Features
- **Shared Albums**: Syncs photos directly from `https://photos.app.goo.gl/...` shared links.
- **Background Sync**: Runs continuously to keep albums up-to-date (configurable interval).
- **Metadata**: Preserves creation timestamps (extracted from Google Photos metadata).
- **Auto-Album**: Automatically creates albums in Immich with the same name as Google Photos (or custom name).
- **Efficient**: Skips already synced photos.

## Configuration

Create a `config.json` in the root directory:

```json
{
    "apiKey": "YOUR_IMMICH_API_KEY",
    "apiURL": "http://your-immich-ip:2283/api",
    "googlePhotos": [
        {
            "url": "https://photos.app.goo.gl/YourAlbumLink1",
            "syncInterval": "12h",
            "albumName": "Vacation 2023"
        },
        {
            "url": "https://photos.app.goo.gl/YourAlbumLink2",
            "syncInterval": "60m"
            // "albumName" omitted -> uses Google Photos album title
        }
    ]
}
```

## Running

1.  **Install Go**: [https://go.dev/doc/install](https://go.dev/doc/install)
2.  **Run**:
    ```bash
    go run main.go
    ```
    Or build and run:
    ```bash
    go build -o immich-sync
    ./immich-sync
    ```

## Environment Variables
Alternatively, you can provide Immich credentials via environment variables:
- `IMMICH_API_KEY`
- `IMMICH_API_URL`

*Note: Google Photos URLs must be configured in `config.json`.*

## License
MIT
