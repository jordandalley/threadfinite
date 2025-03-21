# Threadfinite
Uses the official latest docker image, but applies some optimisations including the following:

- Added 'supervisord' for multi process management
- Added 'nscd' for being able to run ffmpeg static binary builds
- Added 'yt-dlp' for finding optimal stream URL's
- Removed latest buggy ffmpeg builds, and inserts ffmpeg 4.4.1 binary into image at /usr/bin/ffmpeg-4.4.1
- Adds a wrapper script in place of /usr/bin/ffmpeg that adds optimisations for stream fetching and caching capability (See: [threadfin-get-best-stream](https://github.com/jordandalley/threadfin-get-best-stream))

## Installation

1. Download this repository: ```git clone ```
2. Edit the docker-compose.yaml file, to point to your chosen config and tmp paths
3. Build the container: ```docker compose build```
4. Run the container: ```docker compose up -d```
