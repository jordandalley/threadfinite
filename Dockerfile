# start with the official threadfin image
FROM fyb3roptik/threadfin:latest

# remove the ffmpeg package bundled with threadfin, install nscd, yt-dlp, supervisord and python3-pip
RUN apt-get update && apt-get remove ffmpeg -y && apt-get install -y supervisor nscd yt-dlp python3-pip && rm -rf /var/lib/apt/lists/*
# creare run dir required by nscd
RUN mkdir -p /var/run/nscd

# copy supervisord configuration file
COPY build/supervisord.conf /etc/supervisor/supervisord.conf
# copy the wrapper script to /usr/bin/ffmpeg in place of the original ffmpeg
COPY build/ffmpeg-wrapper.py /usr/bin/ffmpeg
# copy the actual ffmpeg binary to /usr/bin/ffmpeg-bin inside the container
COPY build/ffmpeg /usr/bin/ffmpeg-bin
# set the ffmpeg wrapper script as executable
RUN chmod +x /usr/bin/ffmpeg
# set the ffmpeg binary as executable
RUN chmod +x /usr/bin/ffmpeg-bin
# install ffmpeg-python and psutil
RUN pip3 install ffmpeg-python psutil --break-system-packages

# start supervisord
ENTRYPOINT ["/usr/bin/supervisord", "-c", "/etc/supervisor/supervisord.conf"]
