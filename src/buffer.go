// Complete refactor of buffer.go 9-6-2025 by Jordan Dalley
// Original code from Threadfin

package src

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
        "net"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
	"github.com/avfs/avfs/vfs/memfs"
)

func getActiveClientCount() int {
	cleanUpStaleClients()

	total := 0
	BufferInformation.Range(func(key, value interface{}) bool {
		playlist, ok := value.(Playlist)
		if !ok {
			fmt.Printf("Invalid type assertion for playlist: %v\n", value)
			return true
		}

		modified := false

		for clientID, client := range playlist.Clients {
			if client.Connection < 0 {
				fmt.Printf("Client ID %d has negative connections: %d. Resetting to 0.\n", clientID, client.Connection)
				client.Connection = 0
				modified = true
			} else if client.Connection > 1 {
				fmt.Printf("Client ID %d has suspiciously high connections: %d. Resetting to 1.\n", clientID, client.Connection)
				client.Connection = 1
				modified = true
			}

			// Update client map only if changed
			if modified {
				playlist.Clients[clientID] = client
			}

			total += client.Connection
		}

		if modified {
			BufferInformation.Store(key, playlist)
		}

		fmt.Printf("Playlist %s has %d active clients\n", playlist.PlaylistID, len(playlist.Clients))
		return true
	})

	return total
}

func getActivePlaylistCount() int {
	var count int
	BufferInformation.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

func cleanUpStaleClients() {
	BufferInformation.Range(func(key, value interface{}) bool {
		playlist, ok := value.(Playlist)
		if !ok {
			fmt.Printf("Invalid type assertion for playlist: %v\n", value)
			return true
		}

		changed := false
		for clientID, client := range playlist.Clients {
			if client.Connection <= 0 {
				fmt.Printf("Removing stale client ID %d from playlist %s\n", clientID, playlist.PlaylistID)
				delete(playlist.Clients, clientID)
				changed = true
			}
		}

		if changed {
			BufferInformation.Store(key, playlist)
		}

		return true
	})
}

func getClientIP(r *http.Request) string {
	// Check the X-Forwarded-For header first (may contain multiple IPs)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check the X-Real-IP header
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}

	// Fallback to RemoteAddr (remove port if present)
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr may be just an IP without a port
		return strings.TrimSpace(r.RemoteAddr)
	}
	return strings.TrimSpace(ip)
}

func createStreamID(stream map[int]ThisStream, ip, userAgent string) int {
	uniqueIdentifier := fmt.Sprintf("%s-%s", ip, userAgent)

	// First check if this identifier already exists
	for id, s := range stream {
		if s.ClientID == uniqueIdentifier {
			return id
		}
	}

	// Find the first unused stream ID
	for i := 0; ; i++ {
		if _, exists := stream[i]; !exists {
			return i
		}
	}
}

func bufferingStream(
    playlistID string,
    streamingURL string,
    backupStream1, backupStream2, backupStream3 *BackupStream,
    channelName string,
    w http.ResponseWriter,
    r *http.Request,
) {
    time.Sleep(time.Duration(Settings.BufferTimeout) * time.Millisecond)

    w.Header().Set("Connection", "close")
    w.Header().Set("Access-Control-Allow-Origin", "*")

    playlist, streamID, newStream, err := getOrCreatePlaylistAndStream(
        playlistID, streamingURL, backupStream1, backupStream2, backupStream3, channelName, r,
    )
    if err != nil {
        ShowError(err, 000)
        httpStatusError(w, r, 404)
        return
    }

    // If the stream is new and buffering is required, start buffering
    if newStream && !playlist.Streams[streamID].Status {
        stream := playlist.Streams[streamID]
        stream.MD5 = getMD5(streamingURL)
        stream.Folder = playlist.Folder + stream.MD5 + string(os.PathSeparator)
        stream.PlaylistID = playlistID
        stream.PlaylistName = playlist.PlaylistName
        stream.BackupChannel1 = backupStream1
        stream.BackupChannel2 = backupStream2
        stream.BackupChannel3 = backupStream3
        playlist.Streams[streamID] = stream

        Lock.Lock()
        BufferInformation.Store(playlistID, playlist)
        Lock.Unlock()

        if playlist.Buffer == "ffmpeg" {
            go thirdPartyBuffer(streamID, playlistID, false, 0)
        }

        showInfo(fmt.Sprintf(
            "Streaming Status 1:Playlist: %s - Tuner: %d / %d",
            playlist.PlaylistName,
            len(playlist.Streams),
            playlist.Tuner,
        ))

        BufferClients.Store(playlistID+stream.MD5, ClientConnection{Connection: 1})
    }

    w.WriteHeader(200)

    // Start sending buffered content to client
    stream := playlist.Streams[streamID]
    streamMD5Key := playlistID + stream.MD5

    if err := streamToClient(w, r, playlistID, streamID, streamMD5Key); err != nil {
        showDebug(fmt.Sprintf("Buffering error: %v", err), 2)
        killClientConnection(streamID, playlistID, false)
        return
    }
}

func getOrCreatePlaylistAndStream(
    playlistID, streamingURL string,
    backupStream1, backupStream2, backupStream3 *BackupStream,
    channelName string,
    r *http.Request,
) (Playlist, int, bool, error) {
    Lock.Lock()
    defer Lock.Unlock()

    var playlist Playlist
    var streamID int
    var newStream bool = true

    if p, ok := BufferInformation.Load(playlistID); !ok {
        // Playlist does not exist, create it
        playlistType := getPlaylistType(playlistID)
        folder := System.Folder.Temp + playlistID + string(os.PathSeparator)

        if err := checkVFSFolder(folder, bufferVFS); err != nil {
            return Playlist{}, 0, false, err
        }

        playListBuffer := "ffmpeg" // hardcoded buffer type

        playlist = Playlist{
            Folder:        folder,
            PlaylistID:    playlistID,
            Streams:       make(map[int]ThisStream),
            Clients:       make(map[int]ThisClient),
            Buffer:        playListBuffer,
            Tuner:         getTuner(playlistID, playlistType),
            PlaylistName:  getProviderParameter(playlistID, playlistType, "name"),
            HttpProxyIP:   getProviderParameter(playlistID, playlistType, "http_proxy.ip"),
            HttpProxyPort: getProviderParameter(playlistID, playlistType, "http_proxy.port"),
            HttpUserOrigin: getProviderParameter(playlistID, playlistType, "http_headers.origin"),
            HttpUserReferer: getProviderParameter(playlistID, playlistType, "http_headers.referer"),
        }

        streamID = createStreamID(playlist.Streams, getClientIP(r), r.UserAgent())

        client := ThisClient{Connection: 1}

        stream := ThisStream{
            URL:          streamingURL,
            BackupChannel1: backupStream1,
            BackupChannel2: backupStream2,
            BackupChannel3: backupStream3,
            ChannelName:  channelName,
            Status:       false,
        }

        playlist.Streams[streamID] = stream
        playlist.Clients[streamID] = client

        BufferInformation.Store(playlistID, playlist)
        newStream = true

    } else {
        playlist = p.(Playlist)
        // Playlist exists - check if URL is already streaming
        for id, stream := range playlist.Streams {
            if streamingURL == stream.URL {
                client := playlist.Clients[id]
                client.Connection++
                playlist.Clients[id] = client

                // Update backup streams and channel name (refresh)
                stream.BackupChannel1 = backupStream1
                stream.BackupChannel2 = backupStream2
                stream.BackupChannel3 = backupStream3
                stream.ChannelName = channelName
                stream.Status = false
                playlist.Streams[id] = stream

                BufferInformation.Store(playlistID, playlist)

                debug := fmt.Sprintf(
                    "Restream Status:Playlist: %s - Channel: %s - Connections: %d",
                    playlist.PlaylistName, stream.ChannelName, client.Connection,
                )
                showDebug(debug, 1)

                if c, ok := BufferClients.Load(playlistID + stream.MD5); ok {
                    clients := c.(ClientConnection)
                    clients.Connection = client.Connection
                    BufferClients.Store(playlistID+stream.MD5, clients)
                    showInfo(fmt.Sprintf("Streaming Status:Channel: %s (Clients: %d)", stream.ChannelName, clients.Connection))
                }

                streamID = id
                newStream = false
                break
            }
        }

        if newStream {
            // Check tuner limit
            if len(playlist.Streams) >= playlist.Tuner {
                // No new connections available, stream backup if possible (outside this function)
                return playlist, 0, false, fmt.Errorf("tuner limit reached")
            }

            streamID = createStreamID(playlist.Streams, getClientIP(r), r.UserAgent())
            client := ThisClient{Connection: 1}
            stream := ThisStream{
                URL:           streamingURL,
                ChannelName:   channelName,
                Status:        false,
                BackupChannel1: backupStream1,
                BackupChannel2: backupStream2,
                BackupChannel3: backupStream3,
            }

            playlist.Streams[streamID] = stream
            playlist.Clients[streamID] = client

            BufferInformation.Store(playlistID, playlist)
        }
    }
    return playlist, streamID, newStream, nil
}

func getPlaylistType(playlistID string) string {
    switch playlistID[0:1] {
    case "M":
        return "m3u"
    case "H":
        return "hdhr"
    default:
        return "unknown"
    }
}

func streamToClient(w http.ResponseWriter, r *http.Request, playlistID string, streamID int, md5Key string) error {
    timeOut := 0
    streaming := false
    var oldSegments []string

    for {
        p, ok := BufferInformation.Load(playlistID)
        if !ok {
            return fmt.Errorf("playlist info missing")
        }

        playlist := p.(Playlist)
        stream, ok := playlist.Streams[streamID]
        if !ok {
            return fmt.Errorf("stream not found")
        }

        if !stream.Status {
            timeOut++
            time.Sleep(100 * time.Millisecond)

            c, ok := BufferClients.Load(md5Key)
            if !ok {
                return fmt.Errorf("buffer clients missing")
            }

            clients := c.(ClientConnection)
            if clients.Error != nil || (timeOut > 200 && stream.BackupChannel1 == nil && stream.BackupChannel2 == nil && stream.BackupChannel3 == nil) {
                return fmt.Errorf("buffer error or timeout")
            }
            continue
        }

        for {
            // Monitor HTTP client connection
            select {
            case <-r.Context().Done():
                return fmt.Errorf("client connection closed")
            default:
            }

            c, ok := BufferClients.Load(md5Key)
            if !ok {
                return fmt.Errorf("buffer clients missing in inner loop")
            }
            clients := c.(ClientConnection)
            if clients.Error != nil {
                ShowError(clients.Error, 0)
                return fmt.Errorf("client error in buffer clients")
            }

            if _, err := bufferVFS.Stat(stream.Folder); fsIsNotExistErr(err) {
                return fmt.Errorf("buffer folder missing")
            }

            tmpFiles := getBufTmpFiles(&stream)
            if len(tmpFiles) == 0 {
                time.Sleep(100 * time.Millisecond)
                continue
            }

            for _, f := range tmpFiles {
                fileName := stream.Folder + f
                file, err := bufferVFS.Open(fileName)
                if err != nil {
                    return fmt.Errorf("error opening buffer file %s: %w", fileName, err)
                }

                stat, err := file.Stat()
                if err != nil {
                    file.Close()
                    return fmt.Errorf("error stating buffer file %s: %w", fileName, err)
                }

                buffer := make([]byte, stat.Size())
                _, err = file.Read(buffer)
                file.Close()
                if err != nil {
                    return fmt.Errorf("error reading buffer file %s: %w", fileName, err)
                }

                if !streaming {
                    contentType := http.DetectContentType(buffer)
                    w.Header().Set("Content-type", contentType)
                    w.Header().Set("Content-Length", "0")
                    w.Header().Set("Connection", "close")
                }

                if _, err := w.Write(buffer); err != nil {
                    return fmt.Errorf("error writing to client: %w", err)
                }

                streaming = true

                oldSegments = append(oldSegments, f)
                if len(oldSegments) > 20 {
                    fileToRemove := stream.Folder + oldSegments[0]
                    if err := bufferVFS.RemoveAll(getPlatformFile(fileToRemove)); err != nil {
                        ShowError(err, 4007)
                    }
                    oldSegments = oldSegments[1:]
                }
            }
        }
    }
}


func getBufTmpFiles(stream *ThisStream) (tmpFiles []string) {
	tmpFolder := stream.Folder
	var fileIDs []float64

	if _, err := bufferVFS.Stat(tmpFolder); !fsIsNotExistErr(err) {
		files, err := bufferVFS.ReadDir(getPlatformPath(tmpFolder))
		if err != nil {
			ShowError(err, 000)
			return
		}

		// Require at least 3 files? (why 2? - add comment if needed)
		if len(files) > 2 {
			for _, file := range files {
				fileIDStr := strings.TrimSuffix(file.Name(), ".ts")
				fileIDFloat, err := strconv.ParseFloat(fileIDStr, 64)
				if err == nil {
					fileIDs = append(fileIDs, fileIDFloat)
				}
			}

			if len(fileIDs) > 0 {
				sort.Float64s(fileIDs)
				// Remove the last (presumably newest) segment
				fileIDs = fileIDs[:len(fileIDs)-1]

				for _, fileID := range fileIDs {
					fileName := fmt.Sprintf("%d.ts", int64(fileID))

					if indexOfString(fileName, stream.OldSegments) == -1 {
						tmpFiles = append(tmpFiles, fileName)
						stream.OldSegments = append(stream.OldSegments, fileName)
					}
				}
			}
		}
	}

	return
}

func killClientConnection(streamID int, playlistID string, force bool) {
	Lock.Lock()
	defer Lock.Unlock()

	p, ok := BufferInformation.Load(playlistID)
	if !ok {
		return // Playlist not found, nothing to do
	}

	playlist := p.(Playlist)

	if force {
		delete(playlist.Streams, streamID)
		delete(playlist.Clients, streamID)

		if len(playlist.Streams) == 0 {
			BufferInformation.Delete(playlistID)
		} else {
			BufferInformation.Store(playlistID, playlist)
		}

		showInfo(fmt.Sprintf("Streaming Status: Playlist: %s - Tuner: %d / %d", playlist.PlaylistName, len(playlist.Streams), playlist.Tuner))
		return
	}

	stream, streamExists := playlist.Streams[streamID]
	client, clientExists := playlist.Clients[streamID]

	if !streamExists || !clientExists {
		return // Nothing to do if stream or client not found
	}

	cRaw, clientsExist := BufferClients.Load(playlistID + stream.MD5)
	if !clientsExist {
		return
	}

	clients := cRaw.(ClientConnection)

	// Decrement connections
	clients.Connection--
	client.Connection--

	// Clamp to zero
	if clients.Connection < 0 {
		clients.Connection = 0
	}
	if client.Connection < 0 {
		client.Connection = 0
	}

	// Update client and client connection data
	playlist.Clients[streamID] = client
	BufferClients.Store(playlistID+stream.MD5, clients)

	showInfo(fmt.Sprintf("Streaming Status: Channel: %s (Clients: %d)", stream.ChannelName, clients.Connection))

	// Remove stream and clients if no connections remain
	if clients.Connection <= 0 {
		BufferClients.Delete(playlistID + stream.MD5)
		delete(playlist.Streams, streamID)
		delete(playlist.Clients, streamID)
	}

	// Update or delete playlist accordingly
	if len(playlist.Streams) == 0 {
		BufferInformation.Delete(playlistID)
	} else {
		BufferInformation.Store(playlistID, playlist)
		showInfo(fmt.Sprintf("Streaming Status: Playlist: %s - Tuner: %d / %d", playlist.PlaylistName, len(playlist.Streams), playlist.Tuner))
	}
}

func clientConnection(stream ThisStream) (status bool) {
	status = true

	Lock.Lock()
	defer Lock.Unlock()

	// Check if there is a client connection for this stream
	if _, ok := BufferClients.Load(stream.PlaylistID + stream.MD5); !ok {
		showDebug(fmt.Sprintf("Streaming Status: Remove temporary files (%s)", stream.Folder), 1)
		status = false

		showDebug(fmt.Sprintf("Remove tmp folder: %s", stream.Folder), 1)

		if err := bufferVFS.RemoveAll(stream.Folder); err != nil {
			ShowError(err, 4005)
		}

		// Check if the playlist still exists in BufferInformation
		if p, ok := BufferInformation.Load(stream.PlaylistID); ok {
			playlist := p.(Playlist)

			showInfo(fmt.Sprintf("Streaming Status: Channel: %s - No client is using this channel anymore. Streaming Server connection has ended", stream.ChannelName))

			showInfo(fmt.Sprintf("Streaming Status: Playlist: %s - Tuner: %d / %d", playlist.PlaylistName, len(playlist.Streams), playlist.Tuner))

			if len(playlist.Streams) <= 0 {
				BufferInformation.Delete(stream.PlaylistID)
			}
		}

		return status
	}

	return status
}

func switchBandwidth(stream *ThisStream) error {
	if len(stream.DynamicStream) == 0 {
		return errors.New("M3U8 does not contain streaming URLs")
	}

	// Collect and sort available bandwidths
	var bandwidths []int
	for bw := range stream.DynamicStream {
		bandwidths = append(bandwidths, bw)
	}
	sort.Ints(bandwidths)

	var selected DynamicStream

	if stream.NetworkBandwidth == 0 {
		// Default to the lowest bandwidth if no network limit is defined
		selected = stream.DynamicStream[bandwidths[0]]
	} else {
		// Select the highest bandwidth less than or equal to the network limit
		for _, bw := range bandwidths {
			if bw > stream.NetworkBandwidth {
				break
			}
			selected = stream.DynamicStream[bw]
		}
		// Fallback to lowest if no bandwidth is suitable
		if selected.URL == "" {
			selected = stream.DynamicStream[bandwidths[0]]
		}
	}

	// Create the segment
	segment := Segment{
		URL:      selected.URL,
		Duration: 0,
		StreamInf: StreamInf{
			Bandwidth: selected.Bandwidth,
			// Populate additional fields as needed
		},
	}

	stream.Segment = append(stream.Segment, segment)
	return nil
}

// Buffer with FFMPEG
func thirdPartyBuffer(streamID int, playlistID string, useBackup bool, backupNumber int) {
	if p, ok := BufferInformation.Load(playlistID); ok {
		var playlist = p.(Playlist)
		var debug, path, options, bufferType string
		var tmpSegment = 1
		var bufferSize = Settings.BufferSize * 1024
		var stream = playlist.Streams[streamID]
		var buf bytes.Buffer
		var fileSize = 0
		var streamStatus = make(chan bool)

		var tmpFolder = stream.Folder
		var url = stream.URL
		debug = fmt.Sprintf("Buffer FFMpeg Starting: %s", url)
		showDebug(debug, 2)

		if useBackup {
			if backupNumber >= 1 && backupNumber <= 3 {
				switch backupNumber {
				case 1:
					url = stream.BackupChannel1.URL
					showHighlight("START OF BACKUP 1 STREAM")
					showInfo("Backup Channel 1 URL: " + url)
				case 2:
					url = stream.BackupChannel2.URL
					showHighlight("START OF BACKUP 2 STREAM")
					showInfo("Backup Channel 2 URL: " + url)
				case 3:
					url = stream.BackupChannel3.URL
					showHighlight("START OF BACKUP 3 STREAM")
					showInfo("Backup Channel 3 URL: " + url)
				}
			}
		}

		stream.Status = false
		bufferType = strings.ToUpper(playlist.Buffer)

		switch playlist.Buffer {
		case "ffmpeg":
			path = "/home/threadfin/bin/wrapper"
			options = Settings.FFmpegOptions
		default:
			return
		}

		var addErrorToStream = func(err error) {
			if !useBackup || (useBackup && backupNumber >= 0 && backupNumber <= 3) {
				backupNumber++
				if stream.BackupChannel1 != nil || stream.BackupChannel2 != nil || stream.BackupChannel3 != nil {
					thirdPartyBuffer(streamID, playlistID, true, backupNumber)
				}
				return
			}

			if c, ok := BufferClients.Load(playlistID + stream.MD5); ok {
				clients := c.(ClientConnection)
				clients.Error = err
				BufferClients.Store(playlistID+stream.MD5, clients)
			}
		}

		if err := bufferVFS.RemoveAll(getPlatformPath(tmpFolder)); err != nil {
			ShowError(err, 4005)
		}

		if err := checkVFSFolder(tmpFolder, bufferVFS); err != nil {
			ShowError(err, 0)
			showDebug("Buffer Error: checkVFSFolder", 2)
			killClientConnection(streamID, playlistID, false)
			addErrorToStream(err)
			return
		}

		if err := checkFile(path); err != nil {
			ShowError(err, 0)
			showDebug("Buffer Error: checkFile", 2)
			killClientConnection(streamID, playlistID, false)
			addErrorToStream(err)
			return
		}

		showInfo(fmt.Sprintf("%s path:%s", bufferType, path))
		showInfo("Streaming URL:" + url)

		tmpFile := fmt.Sprintf("%s%d.ts", tmpFolder, tmpSegment)
		f, err := bufferVFS.Create(tmpFile)
		f.Close()
		if err != nil {
			ShowError(err, 0)
			showDebug("Buffer Error: bufferVFS.Create", 2)
			killClientConnection(streamID, playlistID, false)
			addErrorToStream(err)
			return
		}

		var args []string
		for i, a := range strings.Split(options, " ") {
			switch bufferType {
			case "FFMPEG":
				a = strings.Replace(a, "[URL]", url, -1)
				if i == 0 {
					if Settings.UserAgent != "" {
						args = []string{"-user_agent", Settings.UserAgent}
					}
					if playlist.HttpProxyIP != "" && playlist.HttpProxyPort != "" {
						args = append(args, "-http_proxy", fmt.Sprintf("http://%s:%s", playlist.HttpProxyIP, playlist.HttpProxyPort))
					}
					headers := ""
					if playlist.HttpUserReferer != "" {
						headers += fmt.Sprintf("Referer: %s\r\n", playlist.HttpUserReferer)
					}
					if playlist.HttpUserOrigin != "" {
						headers += fmt.Sprintf("Origin: %s\r\n", playlist.HttpUserOrigin)
					}
					if headers != "" {
						args = append(args, "-headers", headers)
					}
				}
				args = append(args, a)
			}
		}

		cmd := exec.Command(path, args...)
		cmd.Env = append(os.Environ(), "DISPLAY=:0")
		showDebug(fmt.Sprintf("BUFFER DEBUG: %s:%s %s", bufferType, path, args), 1)

		stdOut, err := cmd.StdoutPipe()
		if err != nil {
			ShowError(err, 0)
			showDebug("Buffer Error: cmd.StdoutPipe", 2)
			killClientConnection(streamID, playlistID, false)
			addErrorToStream(err)
			return
		}

		logOut, err := cmd.StderrPipe()
		if err != nil {
			ShowError(err, 0)
			showDebug("Buffer Error: cmd.StderrPipe", 2)
			killClientConnection(streamID, playlistID, false)
			addErrorToStream(err)
			return
		}

		if buf.Len() == 0 && !stream.Status {
			showInfo(bufferType + ":Processing data")
		}

		cmd.Start()
		defer cmd.Wait()

		go func() {
			scanner := bufio.NewScanner(logOut)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				debug = fmt.Sprintf("%s log:%s", bufferType, strings.TrimSpace(scanner.Text()))
				select {
				case <-streamStatus:
					showDebug(debug, 1)
				default:
					showInfo(debug)
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()

		f, err = bufferVFS.OpenFile(tmpFile, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		buffer := make([]byte, 4096)
		reader := bufio.NewReader(stdOut)
		t := make(chan int)

		go func() {
			timeout := 0
			for {
				time.Sleep(time.Second)
				timeout++
				select {
				case <-t:
					return
				default:
					select {
					case t <- timeout:
					default:
					}
				}
			}
		}()

		for {
			select {
			case timeout := <-t:
				if timeout >= 20 && tmpSegment == 1 {
					showDebug("Buffer Error: Timeout! Killing ffmpeg process!", 2)
					cmd.Process.Kill()
					err = errors.New("Timeout")
					ShowError(err, 4006)
					killClientConnection(streamID, playlistID, false)
					addErrorToStream(err)
					cmd.Wait()
					f.Close()
					return
				}
			default:
			}

			if fileSize == 0 && !stream.Status {
				showInfo("Streaming Status:Receive data from " + bufferType)
			}
			if !clientConnection(stream) {
				showDebug("Buffer Error: No clients for stream. Killing ffmpeg!", 2)
				cmd.Process.Kill()
				f.Close()
				cmd.Wait()
				return
			}

			n, err := reader.Read(buffer)
			if err == io.EOF {
				break
			}
			fileSize += len(buffer[:n])

			if _, err := f.Write(buffer[:n]); err != nil {
				showDebug("Buffer Write Error: Killing ffmpeg!", 2)
				cmd.Process.Kill()
				ShowError(err, 0)
				killClientConnection(streamID, playlistID, false)
				addErrorToStream(err)
				cmd.Wait()
				return
			}

			if fileSize >= bufferSize/2 {
				if tmpSegment == 1 && !stream.Status {
					close(t)
					close(streamStatus)
					showInfo(fmt.Sprintf("Streaming Status:Buffering data from %s", bufferType))
				}
				f.Close()
				tmpSegment++
				if !stream.Status {
					Lock.Lock()
					stream.Status = true
					playlist.Streams[streamID] = stream
					BufferInformation.Store(playlistID, playlist)
					Lock.Unlock()
				}
				tmpFile = fmt.Sprintf("%s%d.ts", tmpFolder, tmpSegment)
				fileSize = 0
				var errCreate, errOpen error

				_, errCreate = bufferVFS.Create(tmpFile)
				f, errOpen = bufferVFS.OpenFile(tmpFile, os.O_APPEND|os.O_WRONLY, 0600)
				if errCreate != nil || errOpen != nil {
					showDebug("Buffer Error: Failed to create file. Killing ffmpeg!", 2)
					cmd.Process.Kill()
					ShowError(errCreate, 0)
					killClientConnection(streamID, playlistID, false)
					addErrorToStream(errCreate)
					cmd.Wait()
					return
				}
			}
		}

		showDebug("Killing ffmpeg process...", 2)
		cmd.Process.Kill()
		cmd.Wait()
		err = errors.New(bufferType + " error")
		addErrorToStream(err)
		ShowError(err, 1204)
		time.Sleep(500 * time.Millisecond)
		clientConnection(stream)
	}
}


func getTuner(id, playlistType string) int {
	tunerStr := getProviderParameter(id, playlistType, "tuner")
	tuner, err := strconv.Atoi(tunerStr)
	if err != nil {
		ShowError(err, 0)
		return 1
	}
	return tuner
}

func initBufferVFS() {
	bufferVFS = memfs.New()
}
