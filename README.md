# Threadfinite
An optimisation of the official [Threadfin](https://github.com/Threadfin/Threadfin) docker image.

## Features

- Optimised highest-quality stream selection
- Prevents orphaned streams and duplicate ffmpeg processes being spawned from Threadfin
- Support for running custom ffmpeg static binaries (eg. less buggy versions)
- Adds better docker process handling with 'supervisord'

## Optimisation Wrapper for FFmpeg

When using proxy mode in threadfin with ffmpeg, it ignores individual stream quality information in the m3u8 manifest and probes all streams to determine which is the highest resolution and quality. This is time consuming and not optimal when the m3u8 manifest contains all the relevant information necessary to determine the best stream.

This python script uses the 'yt-dlp' python library to parse the m3u8 manifest for the highest quality stream (or streams if audio and video separate), then feeds the highest quality stream(s) directly into python-ffmpeg.

If you wish to bypass the optimisation script, and pass the streams directly to ffmpeg like normal, you can simply login to Threadfin and change the ffmpeg binary path from '/usr/bin/ffmpeg' to '/usr/bin/ffmpeg-bin'.

Logs for the wrapper script are stored in config/log and retained for 1 day by default.

## What does the Dockerfile do?

- Adds & removes apt packages in the official Threadfin docker image:
  - python3-pip: added for installing python packages psutil and ffmpeg-python
  - supervisord: added for better process handling, and running nscd alongside the threadfin process
  - nscd: needed for dns resolution of official ffmpeg static builds
  - yt-dlp: used by the ffmpeg wrapper script for parsing m3u manifests
  - ffmpeg: remove this apt package so we can use our own ffmpeg binaries
- Copies the ffmpeg wrapper script in build/ffmpeg-wrapper.py to /usr/bin/ffmpeg
- Copies the supplied ffmped static binary from build/ffmpeg to /usr/bin/ffmpeg-bin
- Installs psutil and ffmpeg-python using pip3

## Installation

1. Download this repository: ```git clone https://github.com/jordandalley/threadfinite.git```
3. Download an ffmpeg binary of your choice from [https://www.johnvansickle.com/ffmpeg/old-releases/](https://www.johnvansickle.com/ffmpeg/old-releases/)
4. Extract the ffmpeg binary into the 'build' directory. Eg. for FFmpeg 4.4.1:
```bash
wget 'https://www.johnvansickle.com/ffmpeg/old-releases/ffmpeg-4.4.1-amd64-static.tar.xz'
tar -xvf ffmpeg-4.4.1-amd64-static.tar.xz -C build/ --strip-components=1 --wildcards '*/ffmpeg'
```
4. Edit the docker-compose.yaml file, to map your volumes to the correct paths for config and tmp directories
5. Build and run the container:
```
docker compose build
docker compose up -d
```
6. Update your Threadfin settings
  - Buffer Size: Start at 0.5MB and work your way up if you have issues
  - Timeout for new client connections: Set this to '0'
  - FFmpeg Binary Path: Set this to "/usr/bin/ffmpeg" if it isn't already set this way

## Script Options

The ffmpeg wrapper script has a few options at the top of the script that can be configured.

There shouldn't normally be any reason to change these defaults, unless running another another environment.

| Variable | Type | Description | Default |
| --- | --- | --- | --- | 
| FFMPEG_PATH | string | Path to the official ffmpeg binary inside the container | "/usr/bin/ffmpeg-bin" |
| LOGGING_ENABLED | boolean | Specifies whether to enable logging | True |
| LOG_RETENTION_DAYS | integer | Specifies the maximum amount of days that log files should be retained for | 1 |
| LOG_DIR | string | Specifies the path in which to store the log files | "/home/threadfin/conf/log" |
| FFMPEG_LOG_LEVEL | string | Specifies the verbosity of ffmpeg logging, if logging is enabled: Valid options include: quiet, info, verbose, and debug | "verbose" |
| PROCESS_CONTROL | boolean | Specifies whether to check for duplicate processes of the same stream, and kills them before continuing | True |
