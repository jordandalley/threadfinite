package src

import (
	"encoding/json"
        "encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"threadfin/src/internal/authentication"
	"github.com/gorilla/websocket"
)

// RequestStruct: Requests over the WebSocket interface
type RequestStruct struct {
        // Commands to Threadfin
        Cmd string `json:"cmd,required"`

        // User
        DeleteUser bool                   `json:"deleteUser,omitempty"`
        UserData   map[string]interface{} `json:"userData,omitempty"`

        // Mapping
        EpgMapping map[string]interface{} `json:"epgMapping,omitempty"`

        // Restore
        Base64 string `json:"base64,omitempty"`

        // New values for the settings (settings.json)
        Settings struct {
                API                      *bool     `json:"api,omitempty"`
                SSDP                     *bool     `json:"ssdp,omitempty"`
                AuthenticationAPI        *bool     `json:"authentication.api,omitempty"`
                AuthenticationM3U        *bool     `json:"authentication.m3u,omitempty"`
                AuthenticationPMS        *bool     `json:"authentication.pms,omitempty"`
                AuthenticationWEP        *bool     `json:"authentication.web,omitempty"`
                AuthenticationXML        *bool     `json:"authentication.xml,omitempty"`
                BackupKeep               *int      `json:"backup.keep,omitempty"`
                BackupPath               *string   `json:"backup.path,omitempty"`
                Buffer                   *string   `json:"buffer,omitempty"`
                BufferSize               *int      `json:"buffer.size.kb,omitempty"`
                BufferTimeout            *float64  `json:"buffer.timeout,omitempty"`
                CacheImages              *bool     `json:"cache.images,omitempty"`
                EpgSource                *string   `json:"epgSource,omitempty"`
                FFmpegOptions            *string   `json:"ffmpeg.options,omitempty"`
                FFmpegPath               *string   `json:"ffmpeg.path,omitempty"`
                FilesUpdate              *bool     `json:"files.update,omitempty"`
                TempPath                 *string   `json:"temp.path,omitempty"`
                Tuner                    *int      `json:"tuner,omitempty"`
                UDPxy                    *string   `json:"udpxy,omitempty"`
                Update                   *[]string `json:"update,omitempty"`
                UserAgent                *string   `json:"user.agent,omitempty"`
                XepgReplaceMissingImages *bool     `json:"xepg.replace.missing.images,omitempty"`
                XepgReplaceChannelTitle  *bool     `json:"xepg.replace.channel.title,omitempty"`
                ThreadfinAutoUpdate      *bool     `json:"ThreadfinAutoUpdate,omitempty"`
                SchemeM3U                *string   `json:"scheme.m3u,omitempty"`
                SchemeXML                *string   `json:"scheme.xml,omitempty"`
                StoreBufferInRAM         *bool     `json:"storeBufferInRAM,omitempty"`
                ForceHttps               *bool     `json:"forceHttps,omitempty"`
                HttpsPort                *int      `json:"httpsPort,omitempty"`
                HttpsThreadfinDomain     *string   `json:"httpsThreadfinDomain,omitempty"`
                HttpThreadfinDomain      *string   `json:"httpThreadfinDomain,omitempty"`
                BindIpAddress            *string   `json:"bindIpAddress,omitempty"`
                EnableNonAscii           *bool     `json:"enableNonAscii,omitempty"`
                EpgCategories            *string   `json:"epgCategories,omitempty"`
                EpgCategoriesColors      *string   `json:"epgCategoriesColors,omitempty"`
                Dummy                    *bool     `json:"dummy,omitempty"`
                DummyChannel             *string   `json:"dummyChannel,omitempty"`
                IgnoreFilters            *bool     `json:"ignoreFilters,omitempty"`
        } `json:"settings,omitempty"`

        // Upload Logo
        Filename string `json:"filename,omitempty"`

        // Filter
        Filter map[int64]interface{} `json:"filter,omitempty"`

        // Files (M3U, HDHR, XMLTV)
        Files struct {
                HDHR  map[string]interface{} `json:"hdhr,omitempty"`
                M3U   map[string]interface{} `json:"m3u,omitempty"`
                XMLTV map[string]interface{} `json:"xmltv,omitempty"`
        } `json:"files,omitempty"`

        // Wizard
        Wizard struct {
                EpgSource *string `json:"epgSource,omitempty"`
                M3U       *string `json:"m3u,omitempty"`
                Tuner     *int    `json:"tuner,omitempty"`
                XMLTV     *string `json:"xmltv,omitempty"`
        } `json:"wizard,omitempty"`

        // Probe Url
        ProbeURL string `json:"probeURL,omitempty"`
}

// ResponseStruct: Responses to the client (WEB)
type ResponseStruct struct {
        ClientInfo struct {
                ARCH           string `json:"arch"`
                Branch         string `json:"branch,omitempty"`
                DVR            string `json:"DVR"`
                EpgSource      string `json:"epgSource"`
                Errors         int    `json:"errors"`
                M3U            string `json:"m3u-url,required"`
                OS             string `json:"os"`
                Streams        string `json:"streams"`
                ActiveClients  int    `json:"activeClients"`
                TotalClients   int    `json:"totalClients"`
                ActivePlaylist int    `json:"activePlaylist"`
                TotalPlaylist  int    `json:"totalPlaylist"`
                UUID           string `json:"uuid"`
                Version        string `json:"version"`
                Warnings       int    `json:"warnings"`
                XEPGCount      int64  `json:"xepg"`
                XML            string `json:"xepg-url,required"`
        } `json:"clientInfo,omitempty"`

        Data struct {
                Playlist struct {
                        M3U struct {
                                Groups struct {
                                        Text  []string `json:"text,required"`
                                        Value []string `json:"value,required"`
                                } `json:"groups,required"`
                        } `json:"m3u,required"`
                } `json:"playlist,required"`

                StreamPreviewUI struct {
                        Active   []string `json:"activeStreams,required"`
                        Inactive []string `json:"inactiveStreams,required"`
                }
        } `json:"data,required"`

        Alert               string                 `json:"alert,omitempty"`
        ConfigurationWizard bool                   `json:"configurationWizard,required"`
        Error               string                 `json:"err,omitempty"`
        Log                 WebScreenLogStruct     `json:"log,required"`
        LogoURL             string                 `json:"logoURL,omitempty"`
        OpenLink            string                 `json:"openLink,omitempty"`
        OpenMenu            string                 `json:"openMenu,omitempty"`
        Reload              bool                   `json:"reload,omitempty"`
        Settings            SettingsStruct         `json:"settings,required"`
        Status              bool                   `json:"status,required"`
        Token               string                 `json:"token,omitempty"`
        Users               map[string]interface{} `json:"users,omitempty"`
        Wizard              int                    `json:"wizard,omitempty"`
        XEPG                map[string]interface{} `json:"xepg,required"`
        ProbeInfo           ProbeInfoStruct        `json:"probeInfo,omitempty"`

        Notification map[string]Notification `json:"notification,omitempty"`
}

type ProbeInfoStruct struct {
        Resolution   string `json:"resolution,omitempty"`
        FrameRate    string `json:"frameRate,omitempty"`
        AudioChannel string `json:"audioChannel,omitempty"`
}

// APIRequestStruct: Request via the API interface
type APIRequestStruct struct {
        Cmd      string `json:"cmd"`
        Password string `json:"password"`
        Token    string `json:"token"`
        Username string `json:"username"`
}

// APIResponseStruct: Response to the client (API)
type APIResponseStruct struct {
        EpgSource        string `json:"epg.source,omitempty"`
        Error            string `json:"err,omitempty"`
        Status           bool   `json:"status,required"`
        StreamsActive    int64  `json:"streams.active,omitempty"`
        StreamsAll       int64  `json:"streams.all,omitempty"`
        StreamsXepg      int64  `json:"streams.xepg,omitempty"`
        Token            string `json:"token,omitempty"`
        URLDvr           string `json:"url.dvr,omitempty"`
        URLM3U           string `json:"url.m3u,omitempty"`
        URLXepg          string `json:"url.xepg,omitempty"`
        VersionAPI       string `json:"version.api,omitempty"`
        VersionThreadfin string `json:"version.threadfin,omitempty"`
}

// WebScreenLogStruct: Logs are stored in RAM and provided for the web interface
type WebScreenLogStruct struct {
        Errors   int      `json:"errors,required"`
        Log      []string `json:"log,required"`
        Warnings int      `json:"warnings,required"`
}


// StartWebserver : Start the Webserver
func StartWebserver() (err error) {
	systemMutex.Lock()
	port := Settings.Port
	ipAddress := System.IPAddress
	if Settings.BindIpAddress != "" {
		ipAddress = Settings.BindIpAddress
	}
	systemMutex.Unlock()

	http.HandleFunc("/", Index)
	http.HandleFunc("/stream/", Stream)
	http.HandleFunc("/xmltv/", Threadfin)
	http.HandleFunc("/m3u/", Threadfin)
	http.HandleFunc("/data/", WS)
	http.HandleFunc("/web/", Web)
	http.HandleFunc("/download/", Download)
	http.HandleFunc("/api/", API)
	http.HandleFunc("/images/", Images)
	http.HandleFunc("/data_images/", DataImages)
	http.HandleFunc("/ppv/enable", enablePPV)
	http.HandleFunc("/ppv/disable", disablePPV)
	http.HandleFunc("/auto/", Auto)

	systemMutex.Lock()
	ips := len(System.IPAddressesV4) + len(System.IPAddressesV6) - 1
	switch ips {
	case 0:
		showHighlight(fmt.Sprintf("Web Interface:%s://%s:%s/web/", System.ServerProtocol.WEB, ipAddress, Settings.Port))
	case 1:
		showHighlight(fmt.Sprintf("Web Interface:%s://%s:%s/web/ | Threadfin is also available via the other %d IP.", System.ServerProtocol.WEB, ipAddress, Settings.Port, ips))
	default:
		showHighlight(fmt.Sprintf("Web Interface:%s://%s:%s/web/ | Threadfin is also available via the other %d IP's.", System.ServerProtocol.WEB, ipAddress, Settings.Port, len(System.IPAddressesV4)+len(System.IPAddressesV6)-1))
	}
	systemMutex.Unlock()

	if err = http.ListenAndServe(ipAddress+":"+port, nil); err != nil {
		ShowError(err, 1001)
		return
	}

	return
}

// Index : Web Server /
func Index(w http.ResponseWriter, r *http.Request) {
	var err error
	var response []byte
	var path = r.URL.Path

	systemMutex.Lock()
	if Settings.HttpThreadfinDomain != "" {
		setGlobalDomain(getBaseUrl(Settings.HttpThreadfinDomain, Settings.Port))
	} else {
		setGlobalDomain(r.Host)
	}
	systemMutex.Unlock()

	switch path {
	case "/discover.json":
		response, err = getDiscover()
		w.Header().Set("Content-Type", "application/json")
	case "/lineup_status.json":
		response, err = getLineupStatus()
		w.Header().Set("Content-Type", "application/json")
	case "/lineup.json":
		systemMutex.Lock()
		if Settings.AuthenticationPMS {
			systemMutex.Unlock()
			_, err := basicAuth(r, "authentication.pms")
			if err != nil {
				ShowError(err, 000)
				httpStatusError(w, r, 403)
				return
			}
		} else {
			systemMutex.Unlock()
		}
		response, err = getLineup()
		w.Header().Set("Content-Type", "application/json")
	case "/device.xml", "/capability":
		response, err = getCapability()
		w.Header().Set("Content-Type", "application/xml")
	default:
		response, err = getCapability()
		w.Header().Set("Content-Type", "application/xml")
	}

	if err == nil {
		w.WriteHeader(200)
		w.Write(response)
		return
	}

	httpStatusError(w, r, 500)
	return
}

// Stream : Web Server /stream/
func Stream(w http.ResponseWriter, r *http.Request) {
	var path = strings.Replace(r.RequestURI, "/stream/", "", 1)
	streamInfo, err := getStreamInfo(path)
	if err != nil {
		ShowError(err, 1203)
		httpStatusError(w, r, 404)
		return
	}

	// If an UDPxy host is set, and the stream URL is multicast (i.e. starts with 'udp://@'),
	// then streamInfo.URL needs to be rewritten to point to UDPxy.
	if Settings.UDPxy != "" && strings.HasPrefix(streamInfo.URL, "udp://@") {
		streamInfo.URL = fmt.Sprintf("http://%s/udp/%s/", Settings.UDPxy, strings.TrimPrefix(streamInfo.URL, "udp://@"))
	}

	systemMutex.Lock()
	forceHttps := Settings.ForceHttps
	systemMutex.Unlock()

	if forceHttps {
		u, err := url.Parse(streamInfo.URL)
		if err == nil {
			u.Scheme = "https"
			hostSplit := strings.Split(u.Host, ":")
			if len(hostSplit) > 0 {
				u.Host = hostSplit[0]
			}
			streamInfo.URL = fmt.Sprintf("https://%s:%d%s?%s", u.Host, Settings.HttpsPort, u.Path, u.RawQuery)
		}
	}

	if r.Method == "HEAD" {
		client := &http.Client{}
		req, err := http.NewRequest("HEAD", streamInfo.URL, nil)
		if err != nil {
			ShowError(err, 1501)
			httpStatusError(w, r, 405)
			return
		}
		resp, err := client.Do(req)
		if err != nil {
			ShowError(err, 1502)
			httpStatusError(w, r, 405)
			return
		}
		defer resp.Body.Close()
		// Copy headers from the source HEAD response to the outgoing response
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		return
	}

	var playListBuffer string
	systemMutex.Lock()
	playListInterface := Settings.Files.M3U[streamInfo.PlaylistID]
	if playListInterface == nil {
		playListInterface = Settings.Files.HDHR[streamInfo.PlaylistID]
	}
        playListBuffer = "ffmpeg"
	systemMutex.Unlock()

	switch playListBuffer {
        case "ffmpeg":
                showInfo(fmt.Sprintf("Buffer:true [%s]", playListBuffer))
        }

	showInfo(fmt.Sprintf("Channel Name:%s", streamInfo.Name))
	showInfo(fmt.Sprintf("Client User-Agent:%s", r.Header.Get("User-Agent")))

	switch playListBuffer {
        case "ffmpeg":
		bufferingStream(streamInfo.PlaylistID, streamInfo.URL, streamInfo.BackupChannel1, streamInfo.BackupChannel2, streamInfo.BackupChannel3, streamInfo.Name, w, r)
        }
	return
}

// Auto: HDHR routing (currently not used)
func Auto(w http.ResponseWriter, r *http.Request) {
	var channelID = strings.Replace(r.RequestURI, "/auto/v", "", 1)
	fmt.Println(channelID)
	return
}

// Threadfin : Web Server /xmltv/ und /m3u/
func Threadfin(w http.ResponseWriter, r *http.Request) {

	var requestType, groupTitle, file, content, contentType string
	var err error
	var path = strings.TrimPrefix(r.URL.Path, "/")
	var groups = []string{}

	systemMutex.Lock()
	if Settings.HttpThreadfinDomain != "" {
		setGlobalDomain(getBaseUrl(Settings.HttpThreadfinDomain, Settings.Port))
	} else {
		setGlobalDomain(r.Host)
	}
	systemMutex.Unlock()

	// XMLTV files
	if strings.Contains(path, "xmltv/") {

		requestType = "xml"

		err = urlAuth(r, requestType)
		if err != nil {
			ShowError(err, 000)
			httpStatusError(w, r, 403)
			return
		}

		systemMutex.Lock()
		file = System.Folder.Data + getFilenameFromPath(path)
		systemMutex.Unlock()

		content, err = readStringFromFile(file)
		if err != nil {
			httpStatusError(w, r, 404)
			return
		}

	}

	// M3U files
	if strings.Contains(path, "m3u/") {

		requestType = "m3u"

		err = urlAuth(r, requestType)
		if err != nil {
			ShowError(err, 000)
			httpStatusError(w, r, 403)
			return
		}

		groupTitle = r.URL.Query().Get("group-title")

		systemMutex.Lock()
		m3uFilePath := System.Folder.Data + "threadfin.m3u"
		systemMutex.Unlock()

		queries := r.URL.Query()
		// Check if the m3u file exists
		if len(queries) == 0 {
			if _, err := os.Stat(m3uFilePath); err == nil {
				log.Println("Serving existing m3u file")
				http.ServeFile(w, r, m3uFilePath)
				return
			}
		}

		log.Println("M3U file does not exist, building new one")

		systemMutex.Lock()
		systemMutex.Unlock()

		if len(groupTitle) > 0 {
			groups = strings.Split(groupTitle, ",")
		}

		content, err = buildM3U(groups)
		if err != nil {
			ShowError(err, 000)
		}

	}

	contentType = http.DetectContentType([]byte(content))
	if strings.Contains(strings.ToLower(contentType), "xml") {
		contentType = "application/xml; charset=utf-8"
	}

	w.Header().Set("Content-Type", contentType)

	if err == nil {
		w.Write([]byte(content))
	}
}

// Images : Image Cache /images/
func Images(w http.ResponseWriter, r *http.Request) {

	var path = strings.TrimPrefix(r.URL.Path, "/")
	systemMutex.Lock()
	filePath := System.Folder.ImagesCache + getFilenameFromPath(path)
	systemMutex.Unlock()

	content, err := readByteFromFile(filePath)
	if err != nil {
		httpStatusError(w, r, 404)
		return
	}

	w.Header().Add("Content-Type", getContentType(filePath))
	w.Header().Add("Content-Length", fmt.Sprintf("%d", len(content)))
	w.WriteHeader(200)
	w.Write(content)

	return
}

// DataImages : Image path for logos / images that have been uploaded /data_images/
func DataImages(w http.ResponseWriter, r *http.Request) {

	var path = strings.TrimPrefix(r.URL.Path, "/")
	systemMutex.Lock()
	filePath := System.Folder.ImagesUpload + getFilenameFromPath(path)
	systemMutex.Unlock()

	content, err := readByteFromFile(filePath)
	if err != nil {
		httpStatusError(w, r, 404)
		return
	}

	w.Header().Add("Content-Type", getContentType(filePath))
	w.Header().Add("Content-Length", fmt.Sprintf("%d", len(content)))
	w.WriteHeader(200)
	w.Write(content)

	return
}

// WS : Web Sockets /ws/
func WS(w http.ResponseWriter, r *http.Request) {

	var request RequestStruct
	var response ResponseStruct
	response.Status = true

	var newToken string

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// Implement any custom origin validation logic here, if needed.
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		ShowError(err, 0)
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		return
	}

	systemMutex.Lock()
	if Settings.HttpThreadfinDomain != "" {
		setGlobalDomain(getBaseUrl(Settings.HttpThreadfinDomain, Settings.Port))
	} else {
		setGlobalDomain(r.Host)
	}
	systemMutex.Unlock()

	for {

		err = conn.ReadJSON(&request)

		if err != nil {
			return
		}

		systemMutex.Lock()
		if System.ConfigurationWizard == false {

			switch Settings.AuthenticationWEB {

			// Token Authentication
			case true:

				var token string
				tokens, ok := r.URL.Query()["Token"]

				if !ok || len(tokens[0]) < 1 {
					token = "-"
				} else {
					token = tokens[0]
				}

				newToken, err = tokenAuthentication(token)
				if err != nil {
					response.Status = false
					response.Reload = true
					response.Error = err.Error()
					request.Cmd = "-"

					if err = conn.WriteJSON(response); err != nil {
						ShowError(err, 1102)
					}

					systemMutex.Unlock()
					return
				}

				response.Token = newToken
				response.Users, _ = authentication.GetAllUserData()

			}

		}
		systemMutex.Unlock()

		switch request.Cmd {
		// Data read commands
		case "getServerConfig":
			// response.Config = Settings

		case "updateLog":
			response = setDefaultResponseData(response, false)
			if err = conn.WriteJSON(response); err != nil {
				ShowError(err, 1022)
			} else {
				return
				break
			}
			return

		// Data write commands
		case "saveSettings":
			var authenticationUpdate = Settings.AuthenticationWEB
			response.Settings, err = updateServerSettings(request)
			if err == nil {
				response.OpenMenu = strconv.Itoa(indexOfString("settings", System.WEB.Menu))

				if Settings.AuthenticationWEB == true && authenticationUpdate == false {
					response.Reload = true
				}

				initBufferVFS()
			}

		case "saveFilesM3U":
			// Reset cache for urls.json
			var filename = getPlatformFile(System.Folder.Config + "urls.json")
			saveMapToJSONFile(filename, make(map[string]StreamInfo))
			Data.Cache.StreamingURLS = make(map[string]StreamInfo)

			err = saveFiles(request, "m3u")
			if err == nil {
				response.OpenMenu = strconv.Itoa(indexOfString("playlist", System.WEB.Menu))
			}
			updateUrlsJson()

		case "updateFileM3U":
			// Reset cache for urls.json
			var filename = getPlatformFile(System.Folder.Config + "urls.json")
			saveMapToJSONFile(filename, make(map[string]StreamInfo))
			Data.Cache.StreamingURLS = make(map[string]StreamInfo)

			err = updateFile(request, "m3u")
			if err == nil {
				response.OpenMenu = strconv.Itoa(indexOfString("playlist", System.WEB.Menu))
				// Rebuild XEPG database to ensure URLs are updated
				err = createXEPGDatabase()
				if err != nil {
					ShowError(err, 000)
					break
				}
				// Update URLs
				updateUrlsJson()
				// Create M3U file to ensure URLs are properly generated
				createM3UFile()
			}

		case "saveFilesHDHR":
			err = saveFiles(request, "hdhr")
			if err == nil {
				response.OpenMenu = strconv.Itoa(indexOfString("playlist", System.WEB.Menu))
			}

		case "updateFileHDHR":
			err = updateFile(request, "hdhr")
			if err == nil {
				response.OpenMenu = strconv.Itoa(indexOfString("playlist", System.WEB.Menu))
			}

		case "saveFilesXMLTV":
			err = saveFiles(request, "xmltv")
			if err == nil {
				response.OpenMenu = strconv.Itoa(indexOfString("xmltv", System.WEB.Menu))
			}

		case "updateFileXMLTV":
			err = updateFile(request, "xmltv")
			if err == nil {
				response.OpenMenu = strconv.Itoa(indexOfString("xmltv", System.WEB.Menu))
			}

		case "saveFilter":
			response.Settings, err = saveFilter(request)
			if err == nil {
				response.OpenMenu = strconv.Itoa(indexOfString("filter", System.WEB.Menu))
			}

		case "saveEpgMapping":
			err = saveXEpgMapping(request)

		case "saveUserData":
			err = saveUserData(request)
			if err == nil {
				response.OpenMenu = strconv.Itoa(indexOfString("users", System.WEB.Menu))
			}

		case "saveNewUser":
			err = saveNewUser(request)
			if err == nil {
				response.OpenMenu = strconv.Itoa(indexOfString("users", System.WEB.Menu))
			}

		case "resetLogs":
			WebScreenLog.Log = make([]string, 0)
			WebScreenLog.Errors = 0
			WebScreenLog.Warnings = 0
			response.OpenMenu = strconv.Itoa(indexOfString("log", System.WEB.Menu))

		case "ThreadfinBackup":
			file, errNew := ThreadfinBackup()
			err = errNew
			if err == nil {
				response.OpenLink = fmt.Sprintf("%s://%s/download/%s", System.ServerProtocol.WEB, System.Domain, file)
			}

		case "ThreadfinRestore":
			WebScreenLog.Log = make([]string, 0)
			WebScreenLog.Errors = 0
			WebScreenLog.Warnings = 0

			if len(request.Base64) > 0 {
				newWebURL, err := ThreadfinRestoreFromWeb(request.Base64)
				if err != nil {
					ShowError(err, 000)
					response.Alert = err.Error()
				}

				if err == nil {
					if len(newWebURL) > 0 {
						response.Alert = "Backup was successfully restored.\nThe port of the sTeVe URL has changed, you have to restart Threadfin.\nAfter a restart, Threadfin can be reached again at the following URL:\n" + newWebURL
					} else {
						response.Alert = "Backup was successfully restored."
						response.Reload = true
					}
					showInfo("Threadfin:" + "Backup successfully restored.")
				}
			}

		case "uploadLogo":
			if len(request.Base64) > 0 {
				response.LogoURL, err = uploadLogo(request.Base64, request.Filename)
				if err == nil {
					if err = conn.WriteJSON(response); err != nil {
						ShowError(err, 1022)
					} else {
						return
					}
				}
			}

		case "saveWizard":
			nextStep, errNew := saveWizard(request)
			err = errNew
			if err == nil {
				if nextStep == 10 {
					System.ConfigurationWizard = false
					response.Reload = true
				} else {
					response.Wizard = nextStep
				}
			}

		case "probeChannel":
			resolution, frameRate, audioChannels, _ := probeChannel(request)
			response.ProbeInfo = ProbeInfoStruct{Resolution: resolution, FrameRate: frameRate, AudioChannel: audioChannels}

		default:
			fmt.Println("+ + + + + + + + + + +", request.Cmd)
		}

		if err != nil {
			response.Status = false
			response.Error = err.Error()
			response.Settings = Settings
		}

		response = setDefaultResponseData(response, true)
		if System.ConfigurationWizard == true {
			response.ConfigurationWizard = System.ConfigurationWizard
		}

		if err = conn.WriteJSON(response); err != nil {
			ShowError(err, 1022)
		} else {
			break
		}

	}

	return
}

// Web : Web Server /web/
func Web(w http.ResponseWriter, r *http.Request) {

	var lang = make(map[string]interface{})
	var err error

	var requestFile = strings.Replace(r.URL.Path, "/web", "html", -1)
	var content, contentType, file string

	var language LanguageUI

	systemMutex.Lock()
	if Settings.HttpThreadfinDomain != "" {
		setGlobalDomain(getBaseUrl(Settings.HttpThreadfinDomain, Settings.Port))
	} else {
		setGlobalDomain(r.Host)
	}
	systemMutex.Unlock()

	systemMutex.Lock()
        systemMutex.Unlock()
        var languageFile = "html/lang/en.json"

        if value, ok := webUI[languageFile].(string); ok {
                content = GetHTMLString(value)
                lang = jsonToMap(content)
        }

	err = json.Unmarshal([]byte(mapToJSON(lang)), &language)
	if err != nil {
		ShowError(err, 000)
		return
	}

	if getFilenameFromPath(requestFile) == "html" {

		systemMutex.Lock()
		if System.ConfigurationWizard == true {
			file = requestFile + "configuration.html"
			Settings.AuthenticationWEB = false
		} else {
			file = requestFile + "index.html"
		}

		if System.ScanInProgress == 1 {
			file = requestFile + "maintenance.html"
		}
		authenticationWebEnabled := Settings.AuthenticationWEB
		systemMutex.Unlock()

		if authenticationWebEnabled == true {
			var username, password, confirm string
			switch r.Method {
			case "POST":
				var allUserData, _ = authentication.GetAllUserData()

				username = r.FormValue("username")
				password = r.FormValue("password")

				if len(allUserData) == 0 {
					confirm = r.FormValue("confirm")
				}

				// First user is created (password confirmation is present)
				if len(confirm) > 0 {

					var token, err = createFirstUserForAuthentication(username, password)
					if err != nil {
						httpStatusError(w, r, 429)
						return
					}
					// Redirect so that the data is cleared from the browser.
					w = authentication.SetCookieToken(w, token)
					http.Redirect(w, r, "/web", 301)
					return

				}

			        // Username and password are present, now being verified
				if len(username) > 0 && len(password) > 0 {

					var token, err = authentication.UserAuthentication(username, password)
					if err != nil {
						file = requestFile + "login.html"
						lang["authenticationErr"] = language.Login.Failed
						break
					}

					w = authentication.SetCookieToken(w, token)
					http.Redirect(w, r, "/web", 301) // Redirect so that the data is cleared from the browser.

				} else {
					w = authentication.SetCookieToken(w, "-")
					http.Redirect(w, r, "/web", 301) // Redirect so that the data is cleared from the browser.
				}

				return

			case "GET":
				lang["authenticationErr"] = ""
				_, token, err := authentication.CheckTheValidityOfTheTokenFromHTTPHeader(w, r)

				if err != nil {
					file = requestFile + "login.html"
					break
				}

				err = checkAuthorizationLevel(token, "authentication.web")
				if err != nil {
					file = requestFile + "login.html"
					break
				}

			}

			allUserData, err := authentication.GetAllUserData()
			if err != nil {
				ShowError(err, 000)
				httpStatusError(w, r, 403)
				return
			}

			systemMutex.Lock()
			if len(allUserData) == 0 && Settings.AuthenticationWEB == true {
				file = requestFile + "create-first-user.html"
			}
			systemMutex.Unlock()
		}

		requestFile = file

		if value, ok := webUI[requestFile]; ok {

			content = GetHTMLString(value.(string))

			if contentType == "text/plain" {
				w.Header().Set("Content-Disposition", "attachment; filename="+getFilenameFromPath(requestFile))
			}

		} else {
			httpStatusError(w, r, 404)
			return
		}

	}

	if value, ok := webUI[requestFile].(string); ok {

		content = GetHTMLString(value)
		contentType = getContentType(requestFile)

		if contentType == "text/plain" {
			w.Header().Set("Content-Disposition", "attachment; filename="+getFilenameFromPath(requestFile))
		}

	} else {
		httpStatusError(w, r, 404)
		return
	}

	contentType = getContentType(requestFile)

	systemMutex.Lock()
	systemMutex.Unlock()

	w.Header().Add("Content-Type", contentType)
	w.WriteHeader(200)

	if contentType == "text/html" || contentType == "application/javascript" {
		content = parseTemplate(content, lang)
	}

	w.Write([]byte(content))
}

// API : API request /api/
func API(w http.ResponseWriter, r *http.Request) {

	if Settings.HttpThreadfinDomain != "" {
		setGlobalDomain(getBaseUrl(Settings.HttpThreadfinDomain, Settings.Port))
	} else {
		setGlobalDomain(r.Host)
	}
	var request APIRequestStruct
	var response APIResponseStruct

	var responseAPIError = func(err error) {

		var response APIResponseStruct

		response.Status = false
		response.Error = err.Error()
		w.Write([]byte(mapToJSON(response)))
		return

	}

	response.Status = true

	if Settings.API == false {
		httpStatusError(w, r, 423)
		return
	}

	if r.Method == "GET" {
		httpStatusError(w, r, 404)
		return
	}

	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		httpStatusError(w, r, 400)
		return

	}

	err = json.Unmarshal(b, &request)
	if err != nil {
		httpStatusError(w, r, 400)
		return
	}

	w.Header().Set("content-type", "application/json")

	if Settings.AuthenticationAPI == true {
		var token string
		switch len(request.Token) {
		case 0:
			if request.Cmd == "login" {
				token, err = authentication.UserAuthentication(request.Username, request.Password)
				if err != nil {
					responseAPIError(err)
					return
				}

			} else {
				err = errors.New("Login incorrect")
				if err != nil {
					responseAPIError(err)
					return
				}

			}

		default:
			token, err = tokenAuthentication(request.Token)
			fmt.Println(err)
			if err != nil {
				responseAPIError(err)
				return
			}

		}
		err = checkAuthorizationLevel(token, "authentication.api")
		if err != nil {
			responseAPIError(err)
			return
		}

		response.Token = token

	}

	switch request.Cmd {
	case "login": // Nothing needs to be passed

	case "status":

		response.VersionThreadfin = System.Version
		response.VersionAPI = System.APIVersion
		response.StreamsActive = int64(len(Data.Streams.Active))
		response.StreamsAll = int64(len(Data.Streams.All))
		response.StreamsXepg = int64(Data.XEPG.XEPGCount)
		response.EpgSource = Settings.EpgSource
		response.URLDvr = System.Domain
		response.URLM3U = System.ServerProtocol.M3U + "://" + System.Domain + "/m3u/threadfin.m3u"
		response.URLXepg = System.ServerProtocol.XML + "://" + System.Domain + "/xmltv/threadfin.xml"

	case "update.m3u":
		err = getProviderData("m3u", "")
		if err != nil {
			break
		}

		err = buildDatabaseDVR()
		if err != nil {
			break
		}

	case "update.hdhr":

		err = getProviderData("hdhr", "")
		if err != nil {
			break
		}

		err = buildDatabaseDVR()
		if err != nil {
			break
		}

	case "update.xmltv":
		err = getProviderData("xmltv", "")
		if err != nil {
			break
		}

	case "update.xepg":
		buildXEPG(false)

	default:
		err = errors.New(getErrMsg(5000))

	}

	if err != nil {
		responseAPIError(err)
	}

	w.Write([]byte(mapToJSON(response)))

	return
}

// Download : File download
func Download(w http.ResponseWriter, r *http.Request) {

	var path = r.URL.Path
	var file = System.Folder.Temp + getFilenameFromPath(path)
	w.Header().Set("Content-Disposition", "attachment; filename="+getFilenameFromPath(file))

	content, err := readStringFromFile(file)
	if err != nil {
		w.WriteHeader(404)
		return
	}

	os.RemoveAll(System.Folder.Temp + getFilenameFromPath(path))
	w.Write([]byte(content))
	return
}

func setDefaultResponseData(response ResponseStruct, data bool) (defaults ResponseStruct) {

	defaults = response

	// Total connections for all playlists
	totalPlaylistCount := 0
	if len(Settings.Files.M3U) > 0 {
		for _, value := range Settings.Files.M3U {

			// Assert that value is a map[string]interface{}
			nestedMap, ok := value.(map[string]interface{})
			if !ok {
				fmt.Printf("Error asserting nested value as map: %v\n", value)
				continue
			}

			// Get the tuner count
			if tuner, exists := nestedMap["tuner"]; exists {
				switch v := tuner.(type) {
				case float64:
					totalPlaylistCount += int(v)
				case int:
					totalPlaylistCount += v
				default:
				}
			}
		}
	}

	// The following data is always passed to the client
	defaults.ClientInfo.ARCH = System.ARCH
	defaults.ClientInfo.EpgSource = Settings.EpgSource
	defaults.ClientInfo.DVR = System.Addresses.DVR
	defaults.ClientInfo.M3U = System.Addresses.M3U
	defaults.ClientInfo.XML = System.Addresses.XML
	defaults.ClientInfo.OS = System.OS
	defaults.ClientInfo.Streams = fmt.Sprintf("%d / %d", len(Data.Streams.Active), len(Data.Streams.All))
	defaults.ClientInfo.UUID = Settings.UUID
	defaults.ClientInfo.Errors = WebScreenLog.Errors
	defaults.ClientInfo.Warnings = WebScreenLog.Warnings
	defaults.ClientInfo.ActiveClients = getActiveClientCount()
	defaults.ClientInfo.ActivePlaylist = getActivePlaylistCount()
	defaults.ClientInfo.TotalClients = Settings.Tuner
	defaults.ClientInfo.TotalPlaylist = totalPlaylistCount
	defaults.Notification = System.Notification
	defaults.Log = WebScreenLog

	switch System.Branch {

	case "master":
		defaults.ClientInfo.Version = fmt.Sprintf("%s", System.Version)

	default:
		defaults.ClientInfo.Version = fmt.Sprintf("%s (%s)", System.Version, System.Build)
		defaults.ClientInfo.Branch = System.Branch

	}

	if data == true {

		defaults.Users, _ = authentication.GetAllUserData()
		//defaults.DVR = System.DVRAddress

		if Settings.EpgSource == "XEPG" {

			defaults.ClientInfo.XEPGCount = Data.XEPG.XEPGCount

			var XEPG = make(map[string]interface{})

			if len(Data.Streams.Active) > 0 {

				XEPG["epgMapping"] = Data.XEPG.Channels
				XEPG["xmltvMap"] = Data.XMLTV.Mapping

			} else {

				XEPG["epgMapping"] = make(map[string]interface{})
				XEPG["xmltvMap"] = make(map[string]interface{})

			}

			defaults.XEPG = XEPG

		}

		defaults.Settings = Settings

		defaults.Data.Playlist.M3U.Groups.Text = Data.Playlist.M3U.Groups.Text
		defaults.Data.Playlist.M3U.Groups.Value = Data.Playlist.M3U.Groups.Value
		defaults.Data.StreamPreviewUI.Active = Data.StreamPreviewUI.Active
		defaults.Data.StreamPreviewUI.Inactive = Data.StreamPreviewUI.Inactive

	}

	return
}

func enablePPV(w http.ResponseWriter, r *http.Request) {
	xepg, err := loadJSONFileToMap(System.File.XEPG)
	if err != nil {
		var response APIResponseStruct

		response.Status = false
		response.Error = err.Error()
		w.Write([]byte(mapToJSON(response)))
	}

	for _, c := range xepg {

		var xepgChannel = c.(map[string]interface{})

		if xepgChannel["x-mapping"] == "PPV" {
			xepgChannel["x-active"] = true
		}
	}

	err = saveMapToJSONFile(System.File.XEPG, xepg)
	if err != nil {
		var response APIResponseStruct

		response.Status = false
		response.Error = err.Error()
		w.Write([]byte(mapToJSON(response)))
		w.WriteHeader(405)
		return
	}
	buildXEPG(false)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
}

func disablePPV(w http.ResponseWriter, r *http.Request) {
	xepg, err := loadJSONFileToMap(System.File.XEPG)
	if err != nil {
		var response APIResponseStruct

		response.Status = false
		response.Error = err.Error()
		w.Write([]byte(mapToJSON(response)))
	}

	for _, c := range xepg {

		var xepgChannel = c.(map[string]interface{})

		if xepgChannel["x-mapping"] == "PPV" && xepgChannel["x-active"] == true {
			xepgChannel["x-active"] = false
		}
	}

	err = saveMapToJSONFile(System.File.XEPG, xepg)
	if err != nil {
		var response APIResponseStruct

		response.Status = false
		response.Error = err.Error()
		w.Write([]byte(mapToJSON(response)))
	}
	buildXEPG(false)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
}

func httpStatusError(w http.ResponseWriter, r *http.Request, httpStatusCode int) {
	http.Error(w, fmt.Sprintf("%s [%d]", http.StatusText(httpStatusCode), httpStatusCode), httpStatusCode)
	return
}

func getContentType(filename string) (contentType string) {

	mimeTypes := map[string]string{
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
		".ico":  "image/x-icon",
		".webp": "image/webp",
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".ogg":  "video/ogg",
		".mp3":  "audio/mp3",
		".wav":  "audio/wav",
	}

	// Extract the file extension and normalize it to lowercase
	ext := strings.ToLower(path.Ext(filename))
	if contentType, exists := mimeTypes[ext]; exists {
		return contentType
	}
	return "text/plain"
}

// GetHTMLString : base64 -> string
func GetHTMLString(base string) string {
	content, _ := base64.StdEncoding.DecodeString(base)
	return string(content)
}
