# Threadfinite
An optimisation of the official [Threadfin](https://github.com/Threadfin/Threadfin) docker image.

## Features and Changes

- Simplified configuration by removing multiple buffering options
  - This fork uses a wrapper to pass stream URL's to yt-dlp, so it detects the best stream before buffering it to FFmpeg.
  - Enables support for using livestream services such as Youtube as channel sources
- Includes jellyfin-ffmpeg7
- Removed auto update options
- Changed container type from Ubuntu to Debian Bookworm (Slim) reducing container size
- Removed the SIGKILL's in Threadfin FFmpeg process management. The wrapper will detect a SIGINT and pass it to FFmpeg to close it gracefully. If unsuccessful, it will only then send a SIGKILL.

## Optimisation Wrapper for FFmpeg

Normally, when using proxy mode in threadfin with ffmpeg, ffmpeg ignores individual stream quality information in the m3u8 manifest and probes all streams to determine which is the highest resolution and quality. This is time consuming and not optimal when the m3u8 manifest contains all the relevant information necessary to determine the best quality stream(s).

This python script uses the 'yt-dlp' python library to parse the m3u8 manifest for the highest quality stream (or streams if audio and video separate), then feeds the highest quality stream(s) directly into python-ffmpeg. This process is also useful for creating custom m3u8 files with livestreams from sources such as Youtube that you may wish to use as a TV channel.

Logs for the wrapper script are stored in config/log and retained for 1 day by default.

## Sample Docker-Compose

Below is a sample docker-compose.yaml file to get you started.

```
services:
  threadfin:
    build: .
    container_name: threadfin
    ports:
      - 34400:34400
    environment:
      #- PUID=1000
      #- PGID=1000
      - TZ=Pacific/Auckland
    volumes:
      - /docker/threadfin/config:/home/threadfin/conf
    restart: unless-stopped
```

## Installation

1. Download this repository: ```git clone https://github.com/jordandalley/threadfinite.git```
2. Edit the docker-compose.yaml file (sample provided above), to map your volumes to the correct paths for config and tmp directories
3. Build and run the container:
```
docker compose build
docker compose up -d
```

## Script options

The wrapper script has a few options that can be configured by putting them into the environmental variables section of the docker-compose.yaml file

There shouldn't normally be any reason to change these defaults, unless running another another environment.

| Variable | Type | Description | Default |
| --- | --- | --- | --- | 
| FFWR_FFMPEG_PATH | string | Path to the official ffmpeg binary inside the container | /usr/bin/ffmpeg-bin |
| FFWR_LOGGING_ENABLED | boolean | Specifies whether to enable logging | True |
| FFWR_LOG_LEVEL | string | Specifies the log level of the script. Valid options: 'NOTSET', 'DEBUG', 'INFO', 'WARNING', 'ERROR', 'CRITICAL' | INFO |
| FFWR_LOG_RETENTION_DAYS | integer | Specifies the maximum amount of days that log files should be retained for | 1 |
| FFWR_LOG_DIR | string | Specifies the path in which to store the log files | /home/threadfin/conf/log |
| FFWR_FFMPEG_LOG_LEVEL | string | Specifies the verbosity of ffmpeg logging, if logging is enabled: Valid options: 'quiet', 'panic', 'fatal', 'error', 'warning', 'debug', 'trace' | verbose |

## Youtube example

If you'd like to use a Youtube livestream as a TV channel, here is an example that may work for you.

In this example, I have created a custom m3u8 file for the International Space Station (ISS) livestream.

```
#EXTM3U
#EXTINF:-1 group-title="Miscellaneous" tvg-logo="https://upload.wikimedia.org/wikipedia/commons/thumb/e/e5/NASA_logo.svg/918px-NASA_logo.svg.png", ISS Livestream
https://www.youtube.com/watch?v=H999s0P1Er0
```

Simply then add this m3u8 file into your config directory, then add it like you would any other source.

![CodeRabbit Pull Request Reviews](https://img.shields.io/coderabbit/prs/github/jordandalley/threadfinite?utm_source=oss&utm_medium=github&utm_campaign=jordandalley%2Fthreadfinite&labelColor=171717&color=FF570A&link=https%3A%2F%2Fcoderabbit.ai&label=CodeRabbit+Reviews)
