# Threadfinite
An optimisation of the official [Threadfin](https://github.com/Threadfin/Threadfin) docker image.

## Features

- Adds optimisations to ffmpeg stream selection and handling
- Adds support for running custom ffmpeg static binaries (eg. less buggy versions)
- Adds better docker process handling with 'supervisord'

## Optimisation Wrapper for FFmpeg

When using ffmpeg in proxy mode in threadfin, ffmpeg ignores individual stream quality information in the m3u8 manifest and probes all streams to determine which is the highest resolution and quality. This is time consuming and not optimal when the m3u8 manifest contains all the relevant information necessary to determine the best stream.

This script passes tne requested stream url to 'yt-dlp' first, which parses the m3u8 manifest for the highest quality stream (or streams if audio and video separate), builds a special ffmpeg command which feeds the highest quality stream directly to it, and caches the command (when appropriate) for subsequent streams.

Cache files are stored in the Threadfin config/cache directory. The cache can be purged by deleting the 'ffcmd-*' files.

If you wish to bypass the optimisation script, and pass the streams directly to ffmpeg like normal, you can simply login to Threadfin and change the ffmpeg binary path from '/usr/bin/ffmpeg' to '/usr/bin/ffmpeg-binary'.

## What does the Dockerfile do?

- Adds & removes apt packages in the official Threadfin docker image:
  - supervisord: added for better process handling, and running nscd alongside the threadfin process
  - nscd: needed for dns resolution of official ffmpeg static builds
  - yt-dlp: used by the ffmpeg wrapper script for parsing m3u manifests
  - ffmpeg: remove this apt package so we can use our own ffmpeg binaries
- Copies the ffmpeg wrapper script in build/ffmpeg_wrapper to /usr/bin/ffmpeg
- Copies the supplied ffmped static binary from build/ffmpeg to /usr/bin/ffmpeg-binary

## Installation

1. Download this repository: ```git clone https://github.com/jordandalley/threadfinite.git```
3. Download an ffmpeg binary of your choice from [https://www.johnvansickle.com/ffmpeg/old-releases/](https://www.johnvansickle.com/ffmpeg/old-releases/)
4. Extract the ffmpeg binary into the 'build' directory (eg. for FFmpeg 4.4.1):
```bash
wget "https://www.johnvansickle.com/ffmpeg/old-releases/ffmpeg-4.4.1-amd64-static.tar.xz"
tar -xvf ffmpeg-4.4.1-amd64-static.tar.xz -C build/ --strip-components=1 --wildcards '*/ffmpeg'
```
4. Edit the docker-compose.yaml file, to map your volumes to the correct paths for config and tmp directories
5. Build and run the container:
```bash
docker compose build
docker compose up -d
```
