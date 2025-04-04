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
import json
import psutil
import threading
import websocket
from datetime import datetime, timedelta

#####################################################################
############## Set these in docker-compose as ENV vars ##############
#####################################################################
# FFmpeg path (Default: '/usr/bin/ffmpeg-bin')
FFMPEG_PATH = os.getenv('FFWR_FFMPEG_PATH','/usr/bin/ffmpeg-bin')
# Specifies whether to enable logging (Default: True)
#LOGGING_ENABLED = os.getenv('FFWR_LOGGING_ENABLED', True).lower() in ('false', '0', 'no')
LOGGING_ENABLED = str(os.getenv('FFWR_LOGGING_ENABLED', 'True')).lower() not in ('false', '0', 'no')
# Amount of days that logs should be retained for (Default: 1)
LOG_RETENTION_DAYS = int(os.getenv('FFWR_LOG_RETENTION_DAYS', '1') or 1)
# Specifies the logging path. This is usually mapped to the host in Docker under the config directory. (Default: "/home/threadfin/conf/log")
LOG_DIR = os.getenv('FFWR_LOG_DIR','/home/threadfin/conf/log')
# Specify the logging verbosity of ffmpeg. (Default: "warning")
FFMPEG_LOG_LEVEL = os.getenv('FFWR_FFMPEG_LOG_LEVEL', 'warning').lower() if os.getenv('LOG_LEVEL', 'warning').lower() in {'quiet', 'panic', 'fatal', 'error', 'warning', 'info', 'verbose', 'debug', 'trace'} else 'warning'
# Specify whether ffmpeg-wrapper should enable process control for ffmpeg (Default: True)
#PROCESS_CONTROL = os.getenv('FFWR_PROCESS_CONTROL', True).lower() in ('false', '0', 'no')
PROCESS_CONTROL = str(os.getenv('FFWR_PROCESS_CONTROL', 'True')).lower() not in ('false', '0', 'no')
# Specifies the interval in seconds at which process control should check the ffmpeg process for activity (Default: 60)
PROCESS_CONTROL_INTERVAL = int(os.getenv('FFWR_PROCESS_CONTROL_INTERVAL', '60') or 60)

def graceful_exit(signal_num, frame):
    """Handler function to handle termination signals gracefully."""
    if signal_num:
        logging.info(f"Received signal: {signal_num}")
    # If the FFmpeg process is running, terminate it
    if ffmpeg_process:
        logging.info("Terminating FFmpeg process")
        ffmpeg_process.terminate()  # Terminate FFmpeg process
        ffmpeg_process.wait()  # Wait for the process to finish
        logging.info("FFmpeg process terminated")
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
        logging.exception(f"An error occurred: {e}")
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
        'loglevel': FFMPEG_LOG_LEVEL,  # Set ffmpeg log level
        'c:v': 'copy',  # Copy the video stream without re-encoding
        'c:a': 'copy',  # Copy the audio stream without re-encoding
        'dn': None, # Don't copy data streams
        'movflags': 'faststart',  # Useful for streaming (moves the moov atom to the beginning of the file)
        'format': 'mpegts',  # Set output format to MPEG-TS
        'fflags': '+genpts+nobuffer',  # Generate PTS (presentation timestamps)
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

        if PROCESS_CONTROL:
            logging.info("Starting Process Control thread")
            # Run the check_ffmpeg_activity function in a separate thread
            monitoring_thread = threading.Thread(target=process_control, args=(ffmpeg_pid,))
            monitoring_thread.daemon = True
            monitoring_thread.start()

        ffmpeg_process.wait()  # Wait for FFmpeg to finish
        logging.info("FFmpeg process has completed.")

    except ffmpeg._run.Error as e:
        logging.error("FFmpeg encountered an error.")
        if e.stderr:
            for line in e.stderr.decode(errors="ignore").splitlines():
                logging.error(line)
        logging.exception(e)

def process_control(ffmpeg_pid,uri="ws://127.0.0.1:34400/data/?Token=undefined"):
    try:
        io_file = f"/proc/{ffmpeg_pid}/io"

        # Check if the process exists initially
        if not os.path.exists(io_file):
            logging.info(f"[Process-Control]: FFmpeg process with PID {ffmpeg_pid} does not exist. Something is broken!")
            graceful_exit(None, None)
            return

        with open(io_file, 'r') as f:
            io_data = f.read()

        # Extract the initial rchar value
        rchar_initial = None
        for line in io_data.splitlines():
            if line.startswith('rchar'):
                # Extract value from the rchar line
                rchar_initial = int(line.split()[1])
                break

        logging.info(f"[Process-Control]: Initial rchar value of ffmpeg PID {ffmpeg_pid}: {rchar_initial}")

        while True:  # Infinite loop to monitor the process indefinitely
            time.sleep(PROCESS_CONTROL_INTERVAL)  # Sleep for seconds specified in PROCESS_CONTROL_INTERVAL
            if not os.path.exists(io_file):
                logging.info(f"[Process-Control]: FFmpeg process with ffmpeg PID {ffmpeg_pid} no longer exists.")
                graceful_exit(None, None)
                return

            with open(io_file, 'r') as f:
                io_data = f.read()

            rchar_current = None
            for line in io_data.splitlines():
                if line.startswith('rchar'):
                    # Extract value from the rchar line
                    rchar_current = int(line.split()[1])
                    break
            logging.info(f"[Process-Control]: Value of rchar for ffmpeg PID {ffmpeg_pid} changed from {rchar_initial} to {rchar_current}")

            if rchar_current == rchar_initial:
                logging.info(f"[Process-Control]: No activity detected in ffmpeg PID {ffmpeg_pid}. Exiting.")
                graceful_exit(None, None)
                return
            else:
                # Update the initial rchar to the current value if activity was detected
                rchar_initial = rchar_current
                # If rchar activity was detected, check to see whether there are any active clients on Threadfin
                logging.info("[Process-Control]: Checking number of active clients")
                active_clients = get_active_clients(uri)
                logging.info(f"[Process-Control]: Current number of active clients: {active_clients}")
                if active_clients == 0:
                    logging.info(f"[Process-Control]: Number of active clients has reached {active_clients}, exiting...")
                    graceful_exit(None, None)
                    return

    except Exception as e:
        logging.error(f"[Process-Control]: An error occurred while monitoring ffmpeg PID {ffmpeg_pid}: {e}")
        graceful_exit(None, None)

def get_active_clients(uri):
    timeout=1
    # turn off noisy websocket logging
    logging.getLogger("websocket").setLevel(logging.CRITICAL)
    # api message through threadfin websocket
    message='{"cmd":"updateLog"}'
    # Variable to store the received data
    received_data = None

    # Function to handle the received message
    def on_message(ws, message):
        nonlocal received_data
        try:
            # Decode the JSON response and store it in a variable
            received_data = json.loads(message)
        except json.JSONDecodeError as e:
            return e
        ws.close()

    # Function to handle the WebSocket opening
    def on_open(ws):
        ws.send(message)  # Send the provided message to the WebSocket server

    # Set up the WebSocket app
    ws = websocket.WebSocketApp(uri, on_message=on_message, on_open=on_open)

    # Start the WebSocket app in a separate thread
    def run_ws():
        ws.run_forever(ping_interval=2, ping_timeout=1)  # Ensure ping_interval > ping_timeout

    websocket_thread = threading.Thread(target=run_ws)
    websocket_thread.start()

    # Wait for the WebSocket to finish or timeout
    start_time = time.time()

    while websocket_thread.is_alive():
        # Check if the timeout has passed
        if time.time() - start_time > timeout:
            ws.close()
            websocket_thread.join()  # Ensure that the thread finishes
            break
        time.sleep(0.1)

    # Return the received data client info for number of active clients
    return received_data["clientInfo"]["activeClients"]

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
        logging.info(f"Process Control Enabled: {PROCESS_CONTROL}")

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
        logging.exception(f"An unexpected error occurred: {e}")
        sys.exit(1)

if __name__ == "__main__":
    # create ffmpeg_command as a global variable
    ffmpeg_process = None
    # Set up signal handling
    signal.signal(signal.SIGINT, graceful_exit)
    signal.signal(signal.SIGTERM, graceful_exit)
    # call main function
    main()
