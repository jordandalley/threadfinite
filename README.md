# Threadfinite
An optimisation of the official [Threadfin](https://github.com/Threadfin/Threadfin) docker image.

## Features

- Optimised highest-quality stream selection
- Enables Threadfin support for using livestream services such as Youtube as channels
- Checks periodically for active client counts within threadfin and exits stuck ffmpeg processes
- Support for running custom ffmpeg static binaries (eg. less regressed versions) by including nscd in the build.
- Adds better docker process handling with 'supervisord'

## Optimisation Wrapper for FFmpeg

Normally, when using proxy mode in threadfin with ffmpeg, ffmpeg ignores individual stream quality information in the m3u8 manifest and probes all streams to determine which is the highest resolution and quality. This is time consuming and not optimal when the m3u8 manifest contains all the relevant information necessary to determine the best quality stream(s).

This python script uses the 'yt-dlp' python library to parse the m3u8 manifest for the highest quality stream (or streams if audio and video separate), then feeds the highest quality stream(s) directly into python-ffmpeg. This process is also useful for creating custom m3u8 files with livestreams from sources such as Youtube that you may wish to use as a TV channel.

If you wish to bypass the optimisation script, and pass the streams directly to ffmpeg like normal, you can simply login to Threadfin and change the ffmpeg binary path from '/usr/bin/ffmpeg' to '/usr/bin/ffmpeg-bin'.

Logs for the wrapper script are stored in config/log and retained for 1 day by default.

## What does the Dockerfile do?

- Adds & removes apt packages in the official Threadfin docker image:
  - Add: python3-pip added for installing python packages
  - Add: supervisord added for better process handling, and running nscd alongside the threadfin process
  - Add: nscd needed for dns resolution of official ffmpeg static builds
  - Remove: ffmpeg remove this apt package so we can use our own ffmpeg binaries
- Copies the ffmpeg wrapper script in build/ffmpeg-wrapper.py to /usr/bin/ffmpeg
- Copies the supplied ffmpeg static binary from build/ffmpeg to /usr/bin/ffmpeg-bin
- Installs all requirements in requirements.txt using pip3
  - Add: ffmpeg-python
  - Add: yt-dlp
  - Add: psutil
  - Add: websocket-client

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
      - FFWR_FFMPEG_PATH=/usr/bin/ffmpeg-bin
      - FFWR_LOGGING_ENABLED=True
      - FFWR_LOG_LEVEL=INFO
      - FFWR_LOG_RETENTION_DAYS=1
      - FFWR_LOG_DIR=/home/threadfin/conf/log
      - FFWR_FFMPEG_LOG_LEVEL=warning
      - FFWR_PROCESS_CONTROL=True
      - FFWR_PROCESS_CONTROL_INTERVAL=60
    volumes:
      - /docker/threadfin/config:/home/threadfin/conf
    restart: unless-stopped
```

## Installation

1. Download this repository: ```git clone https://github.com/jordandalley/threadfinite.git```
3. Download an ffmpeg binary of your choice from [https://www.johnvansickle.com/ffmpeg/](https://www.johnvansickle.com/ffmpeg/)
4. Extract the ffmpeg binary into the 'build' directory. Eg. for the latest FFmpeg release:
```bash
wget 'https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz'
tar -xvf ffmpeg-4.4.1-amd64-static.tar.xz -C build/ --strip-components=1 --wildcards '*/ffmpeg'
```
4. Edit the docker-compose.yaml file, to map your volumes to the correct paths for config and tmp directories
5. Build and run the container:
```
docker compose build
docker compose up -d
```
6. Update your Threadfin settings

![image](https://github.com/user-attachments/assets/bdfae0b5-0ac8-418e-b51d-57b489b3a1c9)

  - Buffer Size: Start at 0.5MB and work your way up if you have buffering problems
  - Timeout for new client connections: Set this to '500'
  - User Agent: Set it to a common UA, such as:
     - Chrome UA: ```Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36```
     - Google DAI, SSAI or SCTE-35 stream issues, try this or something random: ```QuickTime\xaa.7.0.4 (qtver=7.0.4;cpu=PPC;os=Mac 10.3.9)```
  - FFmpeg Binary Path: Set this to "/usr/bin/ffmpeg" if it isn't already set this way


## Script options

The ffmpeg wrapper script has a few options that can be configured by putting them into the environmental variables section of the docker-compose.yaml file

There shouldn't normally be any reason to change these defaults, unless running another another environment.

| Variable | Type | Description | Default |
| --- | --- | --- | --- | 
| FFWR_FFMPEG_PATH | string | Path to the official ffmpeg binary inside the container | /usr/bin/ffmpeg-bin |
| FFWR_LOGGING_ENABLED | boolean | Specifies whether to enable logging | True |
| FFWR_LOG_LEVEL | string | Specifies the log level of the script. Valid options: 'NOTSET', 'DEBUG', 'INFO', 'WARNING', 'ERROR', 'CRITICAL' | INFO |
| FFWR_LOG_RETENTION_DAYS | integer | Specifies the maximum amount of days that log files should be retained for | 1 |
| FFWR_LOG_DIR | string | Specifies the path in which to store the log files | /home/threadfin/conf/log |
| FFWR_FFMPEG_LOG_LEVEL | string | Specifies the verbosity of ffmpeg logging, if logging is enabled: Valid options: 'quiet', 'panic', 'fatal', 'error', 'warning', 'debug', 'trace' | verbose |
| FFWR_PROCESS_CONTROL | boolean | Specifies whether to check run process control, which checks for inactive ffmpeg processes and ensures all processes exit when threadfin active clients is 0 | True |
| FFWR_PROCESS_CONTROL_INTERVAL | integer | Specifies the interval, in seconds in which to run process control | 60 |

## Youtube example

If you'd like to use a Youtube livestream as a TV channel, here is an example that may work for you.

In this example, I have created a custom m3u8 file for the International Space Station (ISS) livestream.

```
#EXTM3U
#EXTINF:-1 group-title="Miscellaneous" tvg-logo="https://upload.wikimedia.org/wikipedia/commons/thumb/e/e5/NASA_logo.svg/918px-NASA_logo.svg.png", ISS Livestream
https://www.youtube.com/watch?v=H999s0P1Er0
```

Simply then add this m3u8 file into your config directory, then add it like you would any other source.

