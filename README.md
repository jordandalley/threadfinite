# Threadfinite
Uses the official latest docker image, but applies some optimisations including the following:

- Adds 'supervisord' for multi process management
- Adds 'nscd' for being able to run ffmpeg static binary builds
- Adds 'yt-dlp' for finding optimal stream URL's
- Removes latest buggy ffmpeg builds using apt-get remove, and inserts an ffmpeg binary of your choice into image at /usr/bin/ffmpeg-binary
- Adds a wrapper script in place of /usr/bin/ffmpeg that adds optimisations for stream fetching and caching capability (See: [threadfin-get-best-stream](https://github.com/jordandalley/threadfin-get-best-stream))

## Installation

1. Download this repository: ```git clone https://github.com/jordandalley/threadfinite.git```
2. Download an ffmpeg binary of your choice from [https://www.johnvansickle.com/ffmpeg/old-releases/](https://www.johnvansickle.com/ffmpeg/old-releases/)
3. Extract the ffmpeg binary into the 'build' directory, eg for ffmpeg 4.4.1:
```
wget "https://www.johnvansickle.com/ffmpeg/old-releases/ffmpeg-4.4.1-amd64-static.tar.xz"
tar xvf ffmpeg-4.4.1-amd64-static.tar.xz -C build/ --strip-components=1 --wildcards '*/ffmpeg'
```
4. Edit the docker-compose.yaml file, to map your volumes to the correct paths for config and tmp
5. Build the container: ```docker compose build```
6. Run the container: ```docker compose up -d```
