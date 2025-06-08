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
import threading
from datetime import datetime, timedelta

#####################################################################
############## Set these in docker-compose as ENV vars ##############
#####################################################################
# FFmpeg path (Default: '/usr/lib/jellyfin-ffmpeg/ffmpeg')
FFMPEG_PATH = os.getenv('FFWR_FFMPEG_PATH','/usr/lib/jellyfin-ffmpeg/ffmpeg')
# Specifies whether to enable logging (Default: True)
LOGGING_ENABLED = str(os.getenv('FFWR_LOGGING_ENABLED', 'True')).lower() not in ('false', '0', 'no')
# Sets the log level for the script. Valid values: 'NOTSET', 'DEBUG', 'INFO', 'WARNING', 'ERROR', 'CRITICAL' (Default: "INFO")
LOG_LEVEL = os.getenv('FFWR_LOG_LEVEL', 'INFO').upper() if os.getenv('LOG_LEVEL', 'INFO').upper() in {'NOTSET', 'DEBUG', 'INFO', 'WARNING', 'ERROR', 'CRITICAL'} else 'INFO'
# Amount of days that logs should be retained for (Default: 1)
LOG_RETENTION_DAYS = int(os.getenv('FFWR_LOG_RETENTION_DAYS', '1'))
# Specifies the logging path. This is usually mapped to the host in Docker under the config directory. (Default: "/home/threadfin/conf/log")
LOG_DIR = os.getenv('FFWR_LOG_DIR','/home/threadfin/conf/log')
# Specify the logging verbosity of ffmpeg. Valid values: 'quiet', 'panic', 'fatal', 'error', 'warning', 'info', 'verbose', 'debug', 'trace' (Default: "warning")
FFMPEG_LOG_LEVEL = os.getenv('FFWR_FFMPEG_LOG_LEVEL', 'warning').lower() if os.getenv('LOG_LEVEL', 'warning').lower() in {'quiet', 'panic', 'fatal', 'error', 'warning', 'info', 'verbose', 'debug', 'trace'} else 'warning'
# FFmpeg initial burst in seconds before returning to 1x readrate (Default: 30)
FFMPEG_INIT_BURST = int(os.getenv('FFWR_FFMPEG_INIT_BURST', '30'))


def graceful_exit(signal_num, frame):
    """Handler function to handle termination signals and other calls gracefully."""
    if signal_num:
        logging.info(f"Received signal: {signal_num}")

    # If the FFmpeg process is running, try to terminate it
    if ffmpeg_process and ffmpeg_process.poll() is None:
        logging.info("Attempting to terminate FFmpeg process gracefully")

        # First attempt to terminate the process
        ffmpeg_process.terminate()

        # Wait for up to 60 seconds for the process to terminate
        start_time = time.time()
        while ffmpeg_process.poll() is None:
            elapsed_time = time.time() - start_time
            if elapsed_time > 60:
                logging.warning("FFmpeg process still not terminated, sending a SIGKILL")
                ffmpeg_process.kill()
                break
            time.sleep(1)

        logging.info("FFmpeg process terminated.")
    logging.info("Exiting")
    sys.exit(0)

def gen_logfile(input_url):
    """Generate a log file"""
    if not LOGGING_ENABLED:
        return None

    # create an md5 hash of the master input url for generating log files
    input_md5 = hashlib.md5(input_url.encode()).hexdigest()

    os.makedirs(LOG_DIR, exist_ok=True)
    timestamp = datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
    log_file = os.path.join(LOG_DIR, f"{input_md5}_{timestamp}.log")

    logging.basicConfig(filename=log_file, level=getattr(logging, LOG_LEVEL), format='%(asctime)s.%(msecs)03d - %(levelname)s - %(message)s', datefmt='%Y-%m-%d %H:%M:%S')
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
        logging.error(f"An error occurred: {e}")
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
    # These are global input arguments that only need to be defined once. Eg. hide_banner, loglevel, threads, max_alloc, protocol_whitelist, protocol_blacklist, probesize, analyzeduration, fpsprobesize etc.
    input_args_global = {
        'hide_banner': None, # hide the ffmpeg banner on startup
        'loglevel': FFMPEG_LOG_LEVEL, # Set ffmpeg log level
    }

    # these input flags are replicated for each input stream
    input_args_url = {
        'user_agent': user_agent, # set user agent against all inputs
        're': None, # set readrate to realtime
        'readrate_initial_burst': FFMPEG_INIT_BURST, # set the initial burst rate in seconds
        'copyts': None, # copy timestamps from each input
    }
    if proxy:
        input_args_url['http_proxy'] = proxy # Add the proxy argument if provided

    output_args = {
        'c:v': 'copy', # Copy the video stream without re-encoding
        'c:a': 'copy', # Copy the audio stream without re-encoding
        'dn': None, # Don't copy data streams
        'mpegts_copyts': '1', # Copy timestamps from inputs into mpegts output
        'format': 'mpegts', # Set output format to mpeg-ts
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
    """Run FFmpeg asynchronously, capture stderr in real-time while letting stdout go to pipe:1."""
    global ffmpeg_process  # Ensure the global variable is accessed

    def log_ffmpeg(ffmpeg_process):
        # Read stderr in real-time and log it as it is produced
        for line in ffmpeg_process.stderr:
            # Decode the byte string to a regular string
            decoded_line = line.decode('utf-8').strip()
            logging.info(f"[ffmpeg]: {decoded_line}")

    try:
        # Log the FFmpeg command to debug
        logging.info(f"Starting FFmpeg process...")

        # Run the FFmpeg command asynchronously with stderr captured
        ffmpeg_process = ffmpeg_command.run_async(
            pipe_stderr=True,   # Capture stderr for logging
            overwrite_output=True,
            cmd=FFMPEG_PATH
        )
        if LOGGING_ENABLED:
            stderr_thread = threading.Thread(target=log_ffmpeg, args=(ffmpeg_process,))
            stderr_thread.daemon = True  # Daemonize the thread so it exits when the main program exits
            stderr_thread.start()

        # get the PID of the ffmpeg process
        ffmpeg_pid = ffmpeg_process.pid

        # Log the PID of the FFmpeg process
        logging.info(f"FFmpeg process started with PID: {ffmpeg_pid}")

        ffmpeg_process.wait()  # Wait for FFmpeg to finish
        logging.info("FFmpeg process has completed.")

    except ffmpeg._run.Error as e:
        logging.error("FFmpeg encountered an error.")
        if e.stderr:
            for line in e.stderr.decode(errors="ignore").splitlines():
                logging.error(line)
        logging.error(e)

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

    # get current pid the script
    script_pid = os.getpid()

    # setup logging and start with general info about the stream. this function will also check if logging is enabled.
    log_file = gen_logfile(input_url)

    if log_file:
        logging.info("Starting ffmpeg-wrapper for Threadfinite: https://github.com/jordandalley/threadfinite")
        logging.info(f"Script PID: {script_pid}")
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
        # construct ffmpeg command using python-ffmpeg
        ffmpeg_command = construct_ffmpeg(urls, user_agent, proxy)
        # run ffmpeg
        ffmpeg_run(ffmpeg_command)
    except Exception as e:
        logging.error(f"An unexpected error occurred: {e}")
        sys.exit(1)

if __name__ == "__main__":
    # create ffmpeg_command as a global variable
    ffmpeg_process = None
    # Set up signal handling
    signal.signal(signal.SIGINT, graceful_exit)
    signal.signal(signal.SIGTERM, graceful_exit)
    # call main function
    main()
