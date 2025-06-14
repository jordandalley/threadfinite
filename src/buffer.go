package src

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
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

type BackupStream struct {
	PlaylistID string
	URL        string
}


// Playlist: Contains all playlist information required by the buffer
type Playlist struct {
	Folder          string
	PlaylistID      string
	PlaylistName    string
	Tuner           int
	HttpProxyIP     string
	HttpProxyPort   string
	HttpUserOrigin  string
	HttpUserReferer string
	Buffer          string

	Clients map[int]ThisClient
	Streams map[int]ThisStream
}

// ThisClient : Clientinfos
type ThisClient struct {
	Connection int
}

// ThisStream: Contains information about the stream to be played from a playlist
type ThisStream struct {
	ChannelName      string
	Error            string
	Folder           string
	MD5              string
	NetworkBandwidth int
	PlaylistID       string
	PlaylistName     string
	Status           bool
	URL              string
	BackupChannel1   *BackupStream
	BackupChannel2   *BackupStream
	BackupChannel3   *BackupStream

	Segment []Segment

	// Server Information
	Location           string
	URLFile            string
	URLHost            string
	URLPath            string
	URLRedirect        string
	URLScheme          string
	URLStreamingServer string

	// Used only for HLS / M3U8
	Body             string
	Difference       float64
	Duration         float64
	DynamicBandwidth bool
	FirstSequence    int64
	HLS              bool
	LastSequence     int64
	M3U8URL          string
	NewSegCount      int
	OldSegCount      int
	Sequence         int64
	TimeDiff         float64
	TimeEnd          time.Time
	TimeSegDuration  float64
	TimeStart        time.Time
	Version          int
	Wait             float64

	DynamicStream map[int]DynamicStream

        // Local temp files
	OldSegments []string

	ClientID string
}

// Segment : URL Segments (HLS / M3U8)
type Segment struct {
	Duration     float64
	Info         bool
	PlaylistType string
	Sequence     int64
	URL          string
	Version      int
	Wait         float64

	StreamInf struct {
		AverageBandwidth int
		Bandwidth        int
		Framerate        float64
		Resolution       string
		SegmentURL       string
	}
}

// DynamicStream: Stream information for dynamic bandwidth
type DynamicStream struct {
	AverageBandwidth int
	Bandwidth        int
	Framerate        float64
	Resolution       string
	URL              string
}

// ClientConnection: Client connections
type ClientConnection struct {
	Connection int
	Error      error
}

// BandwidthCalculation: Bandwidth calculation for the stream
type BandwidthCalculation struct {
	NetworkBandwidth int
	Size             int
	Start            time.Time
	Stop             time.Time
	TimeDiff         float64
}

func getActiveClientCount() (count int) {
	count = 0
	cleanUpStaleClients()

	BufferInformation.Range(func(key, value interface{}) bool {
		playlist, ok := value.(Playlist)
		if !ok {
			fmt.Printf("Invalid type assertion for playlist: %v\n", value)
			return true
		}

		for clientID, client := range playlist.Clients {
			if client.Connection < 0 {
				fmt.Printf("Client ID %d has negative connections: %d. Resetting to 0.\n", clientID, client.Connection)
				client.Connection = 0
				playlist.Clients[clientID] = client
				BufferInformation.Store(key, playlist)
			}
			if client.Connection > 1 {
				fmt.Printf("Client ID %d has suspiciously high connections: %d. Resetting to 1.\n", clientID, client.Connection)
				client.Connection = 1
				playlist.Clients[clientID] = client
				BufferInformation.Store(key, playlist)
			}
			count += client.Connection
		}

		fmt.Printf("Playlist %s has %d active clients\n", playlist.PlaylistID, len(playlist.Clients))
		return true
	})

	return count
}

func getActivePlaylistCount() (count int) {
	count = 0
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

		for clientID, client := range playlist.Clients {
			if client.Connection <= 0 {
				fmt.Printf("Removing stale client ID %d from playlist %s\n", clientID, playlist.PlaylistID)
				delete(playlist.Clients, clientID)
			}
		}
		BufferInformation.Store(key, playlist)
		return true
	})
}

func getClientIP(r *http.Request) string {
	// Check the X-Forwarded-For header first
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// X-Forwarded-For may contain multiple IP addresses; return the first one
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check the X-Real-IP header next
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fallback to RemoteAddr
	ip := r.RemoteAddr
	if strings.Contains(ip, ":") {
		// Remove port if present
		ip = strings.Split(ip, ":")[0]
	}

	return ip
}

func createStreamID(stream map[int]ThisStream, ip, userAgent string) (streamID int) {
	streamID = 0
	uniqueIdentifier := fmt.Sprintf("%s-%s", ip, userAgent)

	for i := 0; i <= len(stream); i++ {
		if _, ok := stream[i]; !ok {
			streamID = i
			break
		}
	}

	if _, ok := stream[streamID]; ok && stream[streamID].ClientID == uniqueIdentifier {
		// Return the same ID if the combination already exists
		return streamID
	}

	return
}

func bufferingStream(playlistID string, streamingURL string, backupStream1 *BackupStream, backupStream2 *BackupStream, backupStream3 *BackupStream, channelName string, w http.ResponseWriter, r *http.Request) {

	time.Sleep(time.Duration(Settings.BufferTimeout) * time.Millisecond)

	var playlist Playlist
	var client ThisClient
	var stream ThisStream
	var streaming = false
	var streamID int
	var debug string
	var timeOut = 0
	var newStream = true

	w.Header().Set("Connection", "close")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Check whether the playlist is already in use
	Lock.Lock()
	if p, ok := BufferInformation.Load(playlistID); !ok {
		Lock.Unlock() // Unlock early if not found
		var playlistType string

		// Playlist is not yet in use, create default values for the playlist
		playlist.Folder = System.Folder.Temp + playlistID + string(os.PathSeparator)
		playlist.PlaylistID = playlistID
		playlist.Streams = make(map[int]ThisStream)
		playlist.Clients = make(map[int]ThisClient)

		err := checkVFSFolder(playlist.Folder, bufferVFS)
		if err != nil {
			ShowError(err, 000)
			httpStatusError(w, r, 404)
			return
		}

		switch playlist.PlaylistID[0:1] {

		case "M":
			playlistType = "m3u"

		case "H":
			playlistType = "hdhr"

		}

		systemMutex.Lock()
		var playListBuffer string
		playListInterface := Settings.Files.M3U[playlistID]
		if playListInterface == nil {
			playListInterface = Settings.Files.HDHR[playlistID]
		}
                // Hard set to ffmpeg
                playListBuffer = "ffmpeg"
		systemMutex.Unlock()

		playlist.Buffer = playListBuffer

		playlist.Tuner = getTuner(playlistID, playlistType)

		playlist.PlaylistName = getProviderParameter(playlist.PlaylistID, playlistType, "name")

		playlist.HttpProxyIP = getProviderParameter(playlist.PlaylistID, playlistType, "http_proxy.ip")
		playlist.HttpProxyPort = getProviderParameter(playlist.PlaylistID, playlistType, "http_proxy.port")

		playlist.HttpUserOrigin = getProviderParameter(playlist.PlaylistID, playlistType, "http_headers.origin")
		playlist.HttpUserReferer = getProviderParameter(playlist.PlaylistID, playlistType, "http_headers.referer")

		// Create default values for the stream
		streamID = createStreamID(playlist.Streams, getClientIP(r), r.UserAgent())

		client.Connection += 1

		stream.URL = streamingURL
		stream.BackupChannel1 = backupStream1
		stream.BackupChannel2 = backupStream2
		stream.BackupChannel3 = backupStream3
		stream.ChannelName = channelName
		stream.Status = false

		playlist.Streams[streamID] = stream
		playlist.Clients[streamID] = client

		Lock.Lock()
		BufferInformation.Store(playlistID, playlist)
		Lock.Unlock()

	} else {
		playlist = p.(Playlist)
		Lock.Unlock()

		// Playlist is already used for streaming
		// Check if the URL is already streaming from another client.
		for id := range playlist.Streams {

			stream = playlist.Streams[id]
			client = playlist.Clients[id]

			stream.BackupChannel1 = backupStream1
			stream.BackupChannel2 = backupStream2
			stream.BackupChannel3 = backupStream3
			stream.ChannelName = channelName
			stream.Status = false

			if streamingURL == stream.URL {

				streamID = id
				newStream = false
				client.Connection += 1

				playlist.Clients[streamID] = client

				Lock.Lock()
				BufferInformation.Store(playlistID, playlist)
				Lock.Unlock()

				debug = fmt.Sprintf("Restream Status:Playlist: %s - Channel: %s - Connections: %d", playlist.PlaylistName, stream.ChannelName, client.Connection)

				showDebug(debug, 1)

				if c, ok := BufferClients.Load(playlistID + stream.MD5); ok {

					var clients = c.(ClientConnection)
					clients.Connection = client.Connection

					showInfo(fmt.Sprintf("Streaming Status:Channel: %s (Clients: %d)", stream.ChannelName, clients.Connection))

					BufferClients.Store(playlistID+stream.MD5, clients)

				}

				break
			}

		}

		// New stream for an already active playlist
		if newStream {

			// Check if the playlist allows another stream (Tuner)
			if len(playlist.Streams) >= playlist.Tuner {
				// If there are backup URLs, use them
				if backupStream1 != nil {
					bufferingStream(backupStream1.PlaylistID, backupStream1.URL, nil, backupStream2, backupStream3, channelName, w, r)
				} else if backupStream2 != nil && backupStream1 == nil {
					bufferingStream(backupStream2.PlaylistID, backupStream2.URL, nil, nil, backupStream3, channelName, w, r)
				} else if backupStream3 != nil && backupStream1 == nil && backupStream2 == nil {
					bufferingStream(backupStream3.PlaylistID, backupStream3.URL, nil, nil, nil, channelName, w, r)
				}

				showInfo(fmt.Sprintf("Streaming Status:Playlist: %s - No new connections available. Tuner = %d", playlist.PlaylistName, playlist.Tuner))

				if value, ok := webUI["html/video/stream-limit.ts"]; ok {

					content := GetHTMLString(value.(string))

					w.WriteHeader(200)
					w.Header().Set("Content-type", "video/mpeg")
					w.Header().Set("Content-Length:", "0")

					for i := 1; i < 60; i++ {
						_ = i
						w.Write([]byte(content))
						time.Sleep(time.Duration(500) * time.Millisecond)
					}

					return
				}

				return
			}

			// Playlist allows another stream (Tuner limit not yet reached)
			// Create default values for the stream
			stream = ThisStream{}
			client = ThisClient{}

			streamID = createStreamID(playlist.Streams, getClientIP(r), r.UserAgent())

			client.Connection = 1
			stream.URL = streamingURL
			stream.ChannelName = channelName
			stream.Status = false
			stream.BackupChannel1 = backupStream1
			stream.BackupChannel2 = backupStream2
			stream.BackupChannel3 = backupStream3

			playlist.Streams[streamID] = stream
			playlist.Clients[streamID] = client

			Lock.Lock()
			BufferInformation.Store(playlistID, playlist)
			Lock.Unlock()

		}

	}

	// Check whether the stream is already being played by another client
	if !playlist.Streams[streamID].Status && newStream {

		// New buffer is needed
		stream = playlist.Streams[streamID]
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

		switch playlist.Buffer {

                case "ffmpeg":
			go thirdPartyBuffer(streamID, playlistID, false, 0)

		default:
			break

		}

		showInfo(fmt.Sprintf("Streaming Status 1:Playlist: %s - Tuner: %d / %d", playlist.PlaylistName, len(playlist.Streams), playlist.Tuner))

		var clients ClientConnection
		clients.Connection = 1
		BufferClients.Store(playlistID+stream.MD5, clients)

	}

	w.WriteHeader(200)

	for { //Loop 1: Wait until the first segment has been downloaded through the buffer

		if p, ok := BufferInformation.Load(playlistID); ok {

			var playlist = p.(Playlist)

			if stream, ok := playlist.Streams[streamID]; ok {

				if !stream.Status {

					timeOut++

					time.Sleep(time.Duration(100) * time.Millisecond)

					if c, ok := BufferClients.Load(playlistID + stream.MD5); ok {

						var clients = c.(ClientConnection)

						if clients.Error != nil || (timeOut > 200 && (playlist.Streams[streamID].BackupChannel1 == nil && playlist.Streams[streamID].BackupChannel2 == nil && playlist.Streams[streamID].BackupChannel3 == nil)) {
                                                        debug = fmt.Sprintf("Buffer Error: Client errpr, killing client connection...")
                                                        showDebug(debug, 2)
							killClientConnection(streamID, stream.PlaylistID, false)
							return
						}

					}

					continue
				}

				var oldSegments []string

				for { // Loop 2: Temporary files are present, data can be sent to the client

					// Monitor HTTP client connection

					ctx := r.Context()
					if ok {

						select {

						case <-ctx.Done():
                                                        debug = fmt.Sprintf("Buffer: cts.done. Killing client connection...")
                                                        showDebug(debug, 2)
							killClientConnection(streamID, playlistID, false)
							return

						default:
							if c, ok := BufferClients.Load(playlistID + stream.MD5); ok {

								var clients = c.(ClientConnection)
								if clients.Error != nil {
                                                                        debug = fmt.Sprintf("Buffer Error: Client Error, killing client connection...")
                                                                        showDebug(debug, 2)
									ShowError(clients.Error, 0)
									killClientConnection(streamID, playlistID, false)
									return
								}

							} else {

								return

							}

						}

					}

					if _, err := bufferVFS.Stat(stream.Folder); fsIsNotExistErr(err) {
                                                debug = fmt.Sprintf("Buffer Error: fsIsNotExistErr, killing client connection...")
                                                showDebug(debug, 2)
						killClientConnection(streamID, playlistID, false)
						return
					}

					var tmpFiles = getBufTmpFiles(&stream)

					for _, f := range tmpFiles {

						if _, err := bufferVFS.Stat(stream.Folder); fsIsNotExistErr(err) {
                                                        debug = fmt.Sprintf("Buffer Error: fsIsNotExistErr, killing client connection...")
                                                        showDebug(debug, 2)
							killClientConnection(streamID, playlistID, false)
							return
						}

						oldSegments = append(oldSegments, f)

						var fileName = stream.Folder + f

						file, err := bufferVFS.Open(fileName)
						if err != nil {
							debug = fmt.Sprintf("Buffer Open (%s)", fileName)
							showDebug(debug, 2)
							return
						}
						defer file.Close()

						if err == nil {

							l, err := file.Stat()
							if err == nil {

								debug = fmt.Sprintf("Buffer Status:Send to client (%s)", fileName)
								showDebug(debug, 2)

								var buffer = make([]byte, int(l.Size()))
								_, err = file.Read(buffer)

								if err == nil {

									file.Seek(0, 0)

									if !streaming {

										contentType := http.DetectContentType(buffer)
										_ = contentType
										w.Header().Set("Content-type", contentType)
										w.Header().Set("Content-Length", "0")
										w.Header().Set("Connection", "close")

									}
									_, err := w.Write(buffer)

									if err != nil {
										file.Close()
										killClientConnection(streamID, playlistID, false)
										return
									}

									file.Close()
									streaming = true

								}

								file.Close()

							}

							var n = indexOfString(f, oldSegments)

							if n > 20 {

								var fileToRemove = stream.Folder + oldSegments[0]
								if err = bufferVFS.RemoveAll(getPlatformFile(fileToRemove)); err != nil {
									ShowError(err, 4007)
								}
								oldSegments = append(oldSegments[:0], oldSegments[0+1:]...)

							}

						}

						file.Close()

					}

					if len(tmpFiles) == 0 {
						time.Sleep(time.Duration(100) * time.Millisecond)
					}

				} // End Loop 2

			} else {

				// Stream not found
				showDebug("Streaming Status:Stream not found. Killing Connection", 3)
				killClientConnection(streamID, stream.PlaylistID, false)
				showInfo(fmt.Sprintf("Streaming Status:Playlist: %s - Tuner: %d / %d", playlist.PlaylistName, len(playlist.Streams), playlist.Tuner))
				return

			}

		} // End BufferInformation

	} // End Loop 1

}

func getBufTmpFiles(stream *ThisStream) (tmpFiles []string) {

	var tmpFolder = stream.Folder
	var fileIDs []float64

	if _, err := bufferVFS.Stat(tmpFolder); !fsIsNotExistErr(err) {

		files, err := bufferVFS.ReadDir(getPlatformPath(tmpFolder))
		if err != nil {
			ShowError(err, 000)
			return
		}

		if len(files) > 2 {

			for _, file := range files {

				var fileID = strings.Replace(file.Name(), ".ts", "", -1)
				var f, err = strconv.ParseFloat(fileID, 64)

				if err == nil {
					fileIDs = append(fileIDs, f)
				}

			}

			sort.Float64s(fileIDs)
			fileIDs = fileIDs[:len(fileIDs)-1]

			for _, file := range fileIDs {

				var fileName = fmt.Sprintf("%d.ts", int64(file))

				if indexOfString(fileName, stream.OldSegments) == -1 {
					tmpFiles = append(tmpFiles, fileName)
					stream.OldSegments = append(stream.OldSegments, fileName)
				}

			}

		}

	}

	return
}

func killClientConnection(streamID int, playlistID string, force bool) {
	Lock.Lock()
	defer Lock.Unlock()

	if p, ok := BufferInformation.Load(playlistID); ok {
		var playlist = p.(Playlist)

		if force {
			delete(playlist.Streams, streamID)
			if len(playlist.Streams) == 0 {
				BufferInformation.Delete(playlistID)
			} else {
				BufferInformation.Store(playlistID, playlist)
			}
			showInfo(fmt.Sprintf("Streaming Status: Playlist: %s - Tuner: %d / %d", playlist.PlaylistName, len(playlist.Streams), playlist.Tuner))
			return
		}

		if stream, ok := playlist.Streams[streamID]; ok {
			client := playlist.Clients[streamID]

			if c, ok := BufferClients.Load(playlistID + stream.MD5); ok {
				var clients = c.(ClientConnection)
				clients.Connection--
				client.Connection--

				// Ensure client connections cannot go below zero
				if client.Connection < 0 {
					client.Connection = 0
				}
				if clients.Connection < 0 {
					clients.Connection = 0
				}

				playlist.Clients[streamID] = client
				BufferClients.Store(playlistID+stream.MD5, clients)

				showInfo(fmt.Sprintf("Streaming Status: Channel: %s (Clients: %d)", stream.ChannelName, clients.Connection))

				if clients.Connection <= 0 {
					BufferClients.Delete(playlistID + stream.MD5)
					delete(playlist.Streams, streamID)
					delete(playlist.Clients, streamID)

					if len(playlist.Streams) == 0 {
						BufferInformation.Delete(playlistID)
					} else {
						BufferInformation.Store(playlistID, playlist)
					}
				} else {
					BufferInformation.Store(playlistID, playlist)
				}

				if len(playlist.Streams) > 0 {
					showInfo(fmt.Sprintf("Streaming Status: Playlist: %s - Tuner: %d / %d", playlist.PlaylistName, len(playlist.Streams), playlist.Tuner))
				}
			}
		}
	}
}

func clientConnection(stream ThisStream) (status bool) {

	status = true
	Lock.Lock()
	defer Lock.Unlock()

	if _, ok := BufferClients.Load(stream.PlaylistID + stream.MD5); !ok {

		var debug = fmt.Sprintf("Streaming Status:Remove temporary files (%s)", stream.Folder)
		showDebug(debug, 1)

		status = false

		debug = fmt.Sprintf("Remove tmp folder:%s", stream.Folder)
		showDebug(debug, 1)

		if err := bufferVFS.RemoveAll(stream.Folder); err != nil {
			ShowError(err, 4005)
		}

		if p, ok := BufferInformation.Load(stream.PlaylistID); !ok {

			showInfo(fmt.Sprintf("Streaming Status:Channel: %s - No client is using this channel anymore. Streaming Server connection has ended", stream.ChannelName))

			if p != nil {
				var playlist = p.(Playlist)

				showInfo(fmt.Sprintf("Streaming Status:Playlist: %s - Tuner: %d / %d", playlist.PlaylistName, len(playlist.Streams), playlist.Tuner))

				if len(playlist.Streams) <= 0 {
					BufferInformation.Delete(stream.PlaylistID)
				}
			}

		}

		status = false

	}

	return
}

func switchBandwidth(stream *ThisStream) (err error) {

	var bandwidth []int
	var dynamicStream DynamicStream
	var segment Segment

	for key := range stream.DynamicStream {
		bandwidth = append(bandwidth, key)
	}

	sort.Ints(bandwidth)

	if len(bandwidth) > 0 {

		for i := range bandwidth {

			segment.StreamInf.Bandwidth = stream.DynamicStream[bandwidth[i]].Bandwidth

			dynamicStream = stream.DynamicStream[bandwidth[0]]

			if stream.NetworkBandwidth == 0 {

				dynamicStream = stream.DynamicStream[bandwidth[0]]
				break

			} else {

				if bandwidth[i] > stream.NetworkBandwidth {
					break
				}

				dynamicStream = stream.DynamicStream[bandwidth[i]]

			}

		}

	} else {

		err = errors.New("M3U8 does not contain streaming URLs")
		return

	}

	segment.URL = dynamicStream.URL
	segment.Duration = 0
	stream.Segment = append(stream.Segment, segment)

	return
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

		var tmpFolder = playlist.Streams[streamID].Folder
		var url = playlist.Streams[streamID].URL
                debug = fmt.Sprintf("Buffer FFMpeg Starting: " + url)
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

                        //hard coding path to python yt-dlp wrapper script for ffmpeg
                        path = "/home/threadfin/bin/wrapper"
			options = Settings.FFmpegOptions

		default:
			return
		}

		var addErrorToStream = func(err error) {
			if !useBackup || (useBackup && backupNumber >= 0 && backupNumber <= 3) {
				backupNumber = backupNumber + 1
				if stream.BackupChannel1 != nil || stream.BackupChannel2 != nil || stream.BackupChannel3 != nil {
					thirdPartyBuffer(streamID, playlistID, true, backupNumber)
				}
				return
			}

			var stream = playlist.Streams[streamID]

			if c, ok := BufferClients.Load(playlistID + stream.MD5); ok {

				var clients = c.(ClientConnection)
				clients.Error = err
				BufferClients.Store(playlistID+stream.MD5, clients)

			}

		}

		if err := bufferVFS.RemoveAll(getPlatformPath(tmpFolder)); err != nil {
			ShowError(err, 4005)
		}

		err := checkVFSFolder(tmpFolder, bufferVFS)
		if err != nil {
			ShowError(err, 0)
                        debug = fmt.Sprintf("Buffer Error: checkVFSFolder")
                        showDebug(debug, 2)
			killClientConnection(streamID, playlistID, false)
			addErrorToStream(err)
			return
		}

		err = checkFile(path)
		if err != nil {
			ShowError(err, 0)
                        debug = fmt.Sprintf("Buffer Error: checkFile")
                        showDebug(debug, 2)
			killClientConnection(streamID, playlistID, false)
			addErrorToStream(err)
			return
		}

		showInfo(fmt.Sprintf("%s path:%s", bufferType, path))
		showInfo("Streaming URL:" + url)

		var tmpFile = fmt.Sprintf("%s%d.ts", tmpFolder, tmpSegment)

		f, err := bufferVFS.Create(tmpFile)
		f.Close()
		if err != nil {
			ShowError(err, 0)
                        debug = fmt.Sprintf("Buffer Error: bufferVFS.Create")
                        showDebug(debug, 2)
			killClientConnection(streamID, playlistID, false)
			addErrorToStream(err)
			return
		}

		// Set User-Agent
		var args []string

		for i, a := range strings.Split(options, " ") {

			switch bufferType {
			case "FFMPEG":
				a = strings.Replace(a, "[URL]", url, -1)
				if i == 0 {
					if len(Settings.UserAgent) != 0 {
						args = []string{"-user_agent", Settings.UserAgent}
					}

					if playlist.HttpProxyIP != "" && playlist.HttpProxyPort != "" {
						args = append(args, "-http_proxy", fmt.Sprintf("http://%s:%s", playlist.HttpProxyIP, playlist.HttpProxyPort))
					}

					var headers string
					if len(playlist.HttpUserReferer) != 0 {
						headers += fmt.Sprintf("Referer: %s\r\n", playlist.HttpUserReferer)
					}
					if len(playlist.HttpUserOrigin) != 0 {
						headers += fmt.Sprintf("Origin: %s\r\n", playlist.HttpUserOrigin)
					}
					if headers != "" {
						args = append(args, "-headers", headers)
					}
				}

				args = append(args, a)

			}

		}

		var cmd = exec.Command(path, args...)
		// Set this explicitly to avoid issues with VLC
		cmd.Env = append(os.Environ(), "DISPLAY=:0")

		debug = fmt.Sprintf("BUFFER DEBUG: %s:%s %s", bufferType, path, args)
		showDebug(debug, 1)

		// Byte data from the process
		stdOut, err := cmd.StdoutPipe()
		if err != nil {
			ShowError(err, 0)
                        debug = fmt.Sprintf("Buffer Error: cmd.StdoutPipe, killing client connection...")
                        showDebug(debug, 2)
			killClientConnection(streamID, playlistID, false)
			addErrorToStream(err)
			return
		}

		// Log data from the process
		logOut, err := cmd.StderrPipe()
		if err != nil {
			ShowError(err, 0)
                        debug = fmt.Sprintf("Buffer Error: cmd.StderrPipe, killing client connection...")
                        showDebug(debug, 2)
			killClientConnection(streamID, playlistID, false)
			addErrorToStream(err)
			return
		}

		if len(buf.Bytes()) == 0 && !stream.Status {
			showInfo(bufferType + ":Processing data")
		}

		cmd.Start()
		defer cmd.Wait()

		go func() {

			// Display log data from the process in debug mode 1.
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

				time.Sleep(time.Duration(10) * time.Millisecond)

			}

		}()

		f, err = bufferVFS.OpenFile(tmpFile, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		buffer := make([]byte, 1024*4)

		reader := bufio.NewReader(stdOut)

		t := make(chan int)

		go func() {

			var timeout = 0
			for {
				time.Sleep(time.Duration(1000) * time.Millisecond)
				timeout++

				select {
				case <-t:
					return
				default:
					// Check if the channel is closed before sending
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
                                        debug = fmt.Sprintf("Buffer Error: Timeout! Killing ffmpeg process!")
                                        showDebug(debug, 2)
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
                                debug = fmt.Sprintf("Buffer Error: No clients for stream. Killing ffmpeg!")
                                showDebug(debug, 2)
				cmd.Process.Kill()
				f.Close()
				cmd.Wait()
				return
			}

			n, err := reader.Read(buffer)
			if err == io.EOF {
				break
			}

			fileSize = fileSize + len(buffer[:n])

			if _, err := f.Write(buffer[:n]); err != nil {
                                debug = fmt.Sprintf("Buffer Write Error: Killing ffmpeg!")
                                showDebug(debug, 2)
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
                                        debug = fmt.Sprintf("Buffer Error: Failed to create file. Killing ffmpeg!")
                                        showDebug(debug, 2)
					cmd.Process.Kill()
					ShowError(err, 0)
					killClientConnection(streamID, playlistID, false)
					addErrorToStream(err)
					cmd.Wait()
					return
				}

			}

		}
                debug = fmt.Sprintf("Killing ffmpeg process...")
                showDebug(debug, 2)

		cmd.Process.Kill()
		cmd.Wait()

		err = errors.New(bufferType + " error")
		addErrorToStream(err)
		ShowError(err, 1204)

		time.Sleep(time.Duration(500) * time.Millisecond)
		clientConnection(stream)

		return

	}

}

func getTuner(id, playlistType string) (tuner int) {

	var playListBuffer string
	systemMutex.Lock()
	playListInterface := Settings.Files.M3U[id]
	if playListInterface == nil {
		playListInterface = Settings.Files.HDHR[id]
	}
        // Hard set to ffmpeg
        playListBuffer = "ffmpeg"
	systemMutex.Unlock()

	switch playListBuffer {

        case "ffmpeg":
		i, err := strconv.Atoi(getProviderParameter(id, playlistType, "tuner"))
		if err == nil {
			tuner = i
		} else {
			ShowError(err, 0)
			tuner = 1
		}

	}

	return
}

func initBufferVFS() {
	bufferVFS = memfs.New()
}

