# Immich Sync

Sync photos from Google Photos Shared Albums to your Immich instance.

## Quick Start (Docker)

1. **Download**
   ```bash
   git clone https://github.com/webgears/immich-sync
   cd immich-sync
   ```

2. **Configure**
   Copy `config.example.json` to `config.json` and add your Immich details and Google Photos links.
   ```bash
   cp config.example.json config.json
   nano config.json
   ```

3. **Run**
   ```bash
   docker-compose up -d
   ```

## Configuration (`config.json`)

### API Permissions
If you are generating a specific API key for this tool, ensure it has the following permissions:
- `user.read`
- `asset.read`
- `asset.create`
- `album.read`
- `album.create`
- `album.update`

Alternatively, you can use a key with "All" permissions.

```json
{
    "apiKey": "YOUR_IMMICH_API_KEY",
    "apiURL": "http://your-immich-ip:2283/api",
    "googlePhotos": [
        {
            "url": "https://photos.app.goo.gl/YourAlbumLink1",
            "syncInterval": "12h",
            "albumName": "Vacation 2023"
        }
    ]
}
```

## Features
- **Shared Albums**: Syncs photos directly from shared links.
- **Efficient**: Streaming uploads with minimal resource usage and no disk writes.
- **Background Sync**: Runs continuously on a schedule.

## Manual Run (Dev)

```bash
go run main.go
```
