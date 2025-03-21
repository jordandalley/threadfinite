# start with the official threadfin image
FROM fyb3roptik/threadfin:latest

# install nscd, yt-dlp and supervisord, and remove the ffmpeg package installed by apt
RUN apt-get update && apt-get install -y supervisor nscd yt-dlp && apt-get remove ffmpeg -y && rm -rf /var/lib/apt/lists/*
# creare run dir for nscd
RUN mkdir -p /var/run/nscd

# copy supervisord configuration file
COPY build/supervisord.conf /etc/supervisor/supervisord.conf
# copy the ffmpeg 4.4.1 binary to /usr/bin inside container
COPY build/ffmpeg-4.4.1 /usr/bin/ffmpeg-4.4.1
# copy the wrapper script in place of ffmpeg
COPY build/ffmpeg_wrapper /usr/bin/ffmpeg
# set the ffmpeg-4.4.1 binary as executable
RUN chmod +x /usr/bin/ffmpeg-4.4.1
# set the ffmpeg wrapper script as executable
RUN chmod +x /usr/bin/ffmpeg

# start supervisord
ENTRYPOINT ["/usr/bin/supervisord", "-c", "/etc/supervisor/supervisord.conf"]
