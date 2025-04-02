#!/usr/bin/env python3

import argparse
import sys
import hashlib
import logging
import os
import io
import signal
import yt_dlp
import ffmpeg
import time
import psutil
from datetime import datetime, timedelta

# FFmpeg path (Default: "/usr/bin/ffmpeg-bin")
FFMPEG_PATH = "/usr/bin/ffmpeg-bin"
# Specifies whether to enable logging (Default: True)
LOGGING_ENABLED = True
# Amount of days that logs should be retained for (Default: 1)
LOG_RETENTION_DAYS = 1
# Specifies the logging path. This is usually mapped to the host in Docker under the config directory. (Default: "/home/threadfin/conf/log")
LOG_DIR = "/home/threadfin/conf/log"
# Specify the logging verbosity of ffmpeg. (Default: "verbose")
FFMPEG_LOG_LEVEL = "verbose"
# Specify whether ffmpeg-wrapper should work to prevent duplicate and orphan processes. (Default: True)
PROCESS_CONTROL = True

def process_control():
    """Check if there's another instance of this script running with the same arguments, and if so, kill it"""
    if not PROCESS_CONTROL:
        return None

    current_pid = os.getpid()
    current_cmdline = sys.argv

    for process in psutil.process_iter(attrs=['pid', 'cmdline']):
        try:
            pid = process.info['pid']
            cmdline = process.info['cmdline']

            if pid != current_pid and cmdline and cmdline == current_cmdline:
                return process
        except (psutil.NoSuchProcess, psutil.AccessDenied):
            continue

    return None

def graceful_exit(signal_num, frame):
    """Handler function to handle termination signals gracefully."""
    print(f"\nReceived signal {signal_num}. Exiting gracefully...")
    sys.exit(0)  # Exit the program with status 0 (success)

def gen_logfile(input_md5):
    """Generate a log file"""
    if not LOGGING_ENABLED:
        return None

    os.makedirs(LOG_DIR, exist_ok=True)
    timestamp = datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
    log_file = os.path.join(LOG_DIR, f"{input_md5}_{timestamp}.log")

    logging.basicConfig(filename=log_file, level=logging.INFO, format='%(asctime)s.%(msecs)03d - %(levelname)s - %(message)s', datefmt='%Y-%m-%d %H:%M:%S')
    return log_file

def clean_old_logs():
    """Remove log files older than the retention period."""
    if not LOGGING_ENABLED:
        return

    cutoff_time = datetime.now() - timedelta(days=LOG_RETENTION_DAYS)
    for log_file in os.listdir(LOG_DIR):
        log_path = os.path.join(LOG_DIR, log_file)
        if os.path.isfile(log_path) and datetime.fromtimestamp(os.path.getmtime(log_path)) < cutoff_time:
            os.remove(log_path)

def get_highest_quality_stream(input_url, user_agent, proxy):
    """Retrieve the stream URL using yt-dlp API with the specified options."""

    ytdl_opts = {
      'quiet': True,
      'simulate': True,
      'forceurl': True,
      'format': 'bv+ba/b',
      'format_sort': ['br'],
      'http_headers': {'User-Agent': user_agent},
      'noprogress': True,
      'proxy': proxy,
    }
    # create a StringIO object to capture stdout because yt-dlp is dumb
    stdout_capture = io.StringIO()
    # save old stdout first
    old_stdout = sys.stdout
    # save current stdout to var
    sys.stdout = stdout_capture

    try:
        with yt_dlp.YoutubeDL(ytdl_opts) as ytdlp:
            # run ytdlp
            ytdlp.download(input_url)

    except Exception as e:
        print(f"An error occurred: {e}")
        return []

    finally:
        # return old stdout to system stdout
        sys.stdout = old_stdout

    # Get the captured stdout as a string
    stdout = stdout_capture.getvalue()
    # sanitise the output by splitting each URL out
    urls = stdout.splitlines()
    # strip any dodgy chars
    output = [url.strip() for url in urls]
    # return a clean list of urls
    return output

def construct_ffmpeg(urls, user_agent, proxy):
    """Construct the FFmpeg process based on the retrieved URLs."""
    input_args_global = {
        'fflags': '+discardcorrupt+genpts',
        'analyzeduration': '3000000',
        'probesize': '10M'
    }
    input_args_url = {
        'user_agent': user_agent,
        'thread_queue_size': '10000'
    }

    if proxy:
        input_args_url['http_proxy'] = proxy  # Add the proxy argument if provided

    output_args = {
        'loglevel': FFMPEG_LOG_LEVEL,
        'c:v': 'copy',  # Copy the video stream without re-encoding
        'c:a': 'copy',  # Copy the audio stream without re-encoding
        'dn': None, # Don't copy data streams
        'movflags': 'faststart',  # Useful for streaming (moves the moov atom to the beginning of the file)
        'format': 'mpegts',  # Set output format to MPEG-TS
        'fflags': '+genpts'  # Generate PTS (presentation timestamps)
    }
    ffmpeg_input = []
    ffmpeg_input.append(ffmpeg.input(urls[0], **input_args_global, **input_args_url))
    if len(urls) > 1:
        ffmpeg_input.append(ffmpeg.input(urls[1], **input_args_url))

    # create outputs including pipe to stdout
    ffmpeg_command = ffmpeg.output(*ffmpeg_input, 'pipe:1', **output_args)

    # Log the constructed FFmpeg command for debugging
    logging.info(f"FFmpeg Command: {' '.join(ffmpeg_command.compile())}")

    # Return the constructed FFmpeg command
    return ffmpeg_command

def ffmpeg_run(ffmpeg_command):
    """Run the constructed FFmpeg command and log stderr to logging.info."""
    try:
        # Execute the FFmpeg command with stdout and stderr captured
        out, err = ffmpeg_command.run(capture_stderr=True, overwrite_output=True, cmd=FFMPEG_PATH)
        # Log stderr output from FFmpeg using logging.info
        for line in err.decode().splitlines():
            logging.info(line)
        logging.info("FFmpeg process executed successfully.")
    except ffmpeg.Error as e:
        # Capture stderr and stdout if FFmpeg fails
        for line in e.stderr.decode().splitlines():
            logging.info(line)
        logging.error(f"An error occurred: {e}")

def main():
    # initialise vars
    filtered_args = []
    skip_next = False

    # use argvars to parse input options. Ignore everything except for -i, -user_agent and -http_proxy in any order
    for i, arg in enumerate(sys.argv[1:]):
        if skip_next:
            skip_next = False
            continue
        if arg in ("-i", "-user_agent", "-http_proxy"):
            filtered_args.append(arg)
            if i + 1 < len(sys.argv) - 1:
                filtered_args.append(sys.argv[i + 2])
                skip_next = True
    sys.argv = [sys.argv[0]] + filtered_args
    parser = argparse.ArgumentParser(description="FFmpeg Wrapper Script", allow_abbrev=False)
    parser.add_argument("-i", required=True, help="Specify the input URL")
    parser.add_argument("-user_agent", required=True, help="Specify the User-Agent string")
    parser.add_argument("-http_proxy", help="Specify an HTTP proxy to use (e.g., 'http://proxy.server.address:3128')")
    args, _ = parser.parse_known_args()

    # set the args into vars
    input_url = args.i
    user_agent = args.user_agent
    proxy = args.http_proxy

    # create an md5 hash of the master input url for generating log files
    input_md5 = hashlib.md5(input_url.encode()).hexdigest()

    # setup logging and start with general info about the stream. this function will also check if logging is enabled.
    log_file = gen_logfile(input_md5)

    if log_file:
        logging.info(f"Master URL: {input_url}")
        logging.info(f"User Agent: {user_agent}")
        if proxy:
            logging.info(f"Proxy Server: {proxy}")
        logging.info(f"Log Retention: {LOG_RETENTION_DAYS} day(s)")
        logging.info(f"FFmpeg Log Level: {FFMPEG_LOG_LEVEL}")
        logging.info(f"Cleaning logs older than {LOG_RETENTION_DAYS} days")

        # clean up old logs
        clean_old_logs()

    try:
        logging.info("Finding highest quality stream...")
        # fetch highest quality urls using yt-dlp python library
        urls = get_highest_quality_stream(input_url, user_agent, proxy)
        logging.info(f"Found the following url(s): {urls}")
        logging.info("Constructing FFmpeg command and running...")
        # construct ffmpeg command using python-ffmpeg
        ffmpeg_command = construct_ffmpeg(urls, user_agent, proxy)
        # run ffmpeg
        ffmpeg_run(ffmpeg_command)
    except Exception as e:
        logging.exception(f"An unexpected error occurred: {e}")
        sys.exit(1)

if __name__ == "__main__":
    # First do process control by checking for a lock file associated with the input url md5 hash
    process_control()
    # Set up signal handling
    signal.signal(signal.SIGINT, graceful_exit)
    signal.signal(signal.SIGTERM, graceful_exit)
    # call main function
    main()
