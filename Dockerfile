# start with the official threadfin image
FROM fyb3roptik/threadfin:latest

# install nscd, yt-dlp and supervisord, and remove the ffmpeg package installed by apt
RUN apt-get update && apt-get install -y supervisor nscd yt-dlp && apt-get remove ffmpeg -y && rm -rf /var/lib/apt/lists/*
# creare run dir for nscd
RUN mkdir -p /var/run/nscd

# copy supervisord configuration file
COPY build/supervisord.conf /etc/supervisor/supervisord.conf
# copy the wrapper script to /usr/bin in place of the original ffmpeg
COPY build/ffmpeg-wrapper /usr/bin/ffmpeg
# copy the ffmpeg binary to /usr/bin/ffmpeg-binary inside container
COPY build/ffmpeg /usr/bin/ffmpeg-binary
# set the ffmpeg wrapper script as executable
RUN chmod +x /usr/bin/ffmpeg
# set the ffmpeg-binary file as executable
RUN chmod +x /usr/bin/ffmpeg-binary

# start supervisord
ENTRYPOINT ["/usr/bin/supervisord", "-c", "/etc/supervisor/supervisord.conf"]
