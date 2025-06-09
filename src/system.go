package src

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"time"
	"threadfin/src/internal/imgcache"
)

// SystemStruct: Contains all system information
type SystemStruct struct {
        Addresses struct {
                DVR string
                M3U string
                XML string
        }

        APIVersion             string
        AppName                string
        ARCH                   string
        BackgroundProcess      bool
        Branch                 string
        Build                  string
        Compatibility          string
        ConfigurationWizard    bool
        DBVersion              string
        Dev                    bool
        DeviceID               string
        Domain                 string
        PlexChannelLimit       int
        UnfilteredChannelLimit int

        FFmpeg struct {
                DefaultOptions string
                Path           string
        }

        File struct {
                Authentication string
                M3U            string
                PMS            string
                Settings       string
                URLS           string
                XEPG           string
                XML            string
        }

        Compressed struct {
                GZxml string
        }

        Flag struct {
                Branch  string
                Debug   int
                Info    bool
                Port    string
                Restore string
                SSDP    bool
        }

        Folder struct {
                Backup       string
                Cache        string
                Config       string
                Data         string
                ImagesCache  string
                ImagesUpload string
                Temp         string
        }

        Hostname               string
        ImageCachingInProgress int
        IPAddress              string
        IPAddressesList        []string
        IPAddressesV4          []string
        IPAddressesV6          []string
        Name                   string
        OS                     string
        ScanInProgress         int
        TimeForAutoUpdate      string

        Notification map[string]Notification

        ServerProtocol struct {
                API string
                DVR string
                M3U string
                WEB string
                XML string
        }

        /*GitHub struct {
                Branch  string
                Repo    string
                Update  bool
                User    string
                TagName string
        }

        Update struct {
                Git    string
                Name   string
                Github string
        }*/

        URLBase string
        UDPxy   string
        Version string
        WEB     struct {
                Menu []string
        }
}

// DataStruct: All data is stored here (Lineup, XMLTV)
type DataStruct struct {
        Cache struct {
                Images      *imgcache.Cache
                ImagesCache []string
                ImagesFiles []string
                ImagesURLS  []string
                PMS         map[string]string

                StreamingURLS map[string]StreamInfo
                XMLTV         map[string]XMLTV

                Streams struct {
                        Active []string
                }
        }

        Filter []Filter

        Playlist struct {
                M3U struct {
                        Groups struct {
                                Text  []string
                                Value []string
                        }
                }
        }

        StreamPreviewUI struct {
                Active   []string
                Inactive []string
        }

        Streams struct {
                Active   []interface{}
                All      []interface{}
                Inactive []interface{}
        }

        XMLTV struct {
                Files   []string
                Mapping map[string]interface{}
        }

        XEPG struct {
                Channels         map[string]interface{}
                ChannelsFiltered map[string]interface{}
                XEPGCount        int64
        }
}

// Filter: Used for filter rules
type Filter struct {
        CaseSensitive bool
        LiveEvent     bool
        Rule          string
        Type          string
}

// XEPGChannelStruct : XEPG Structure
type XEPGChannelStruct struct {
        FileM3UID          string        `json:"_file.m3u.id"`
        FileM3UName        string        `json:"_file.m3u.name"`
        FileM3UPath        string        `json:"_file.m3u.path"`
        GroupTitle         string        `json:"group-title"`
        Name               string        `json:"name"`
        TvgID              string        `json:"tvg-id"`
        TvgLogo            string        `json:"tvg-logo"`
        TvgName            string        `json:"tvg-name"`
        TvgChno            string        `json:"tvg-chno"`
        URL                string        `json:"url"`
        UUIDKey            string        `json:"_uuid.key"`
        UUIDValue          string        `json:"_uuid.value,omitempty"`
        Values             string        `json:"_values"`
        XActive            bool          `json:"x-active"`
        XCategory          string        `json:"x-category"`
        XChannelID         string        `json:"x-channelID"`
        XEPG               string        `json:"x-epg"`
        XGroupTitle        string        `json:"x-group-title"`
        XMapping           string        `json:"x-mapping"`
        XmltvFile          string        `json:"x-xmltv-file"`
        XPpvExtra          string        `json:"x-ppv-extra"`
        XBackupChannel1    string        `json:"x-backup-channel-1"`
        XBackupChannel2    string        `json:"x-backup-channel-2"`
        XBackupChannel3    string        `json:"x-backup-channel-3"`
        XHideChannel       bool          `json:"x-hide-channel"`
        XName              string        `json:"x-name"`
        XUpdateChannelIcon bool          `json:"x-update-channel-icon"`
        XUpdateChannelName bool          `json:"x-update-channel-name"`
        XDescription       string        `json:"x-description"`
        Live               bool          `json:"live"`
        IsBackupChannel    bool          `json:"is_backup_channel"`
        BackupChannel1     *BackupStream `json:"backup_channel_1"`
        BackupChannel2     *BackupStream `json:"backup_channel_2"`
        BackupChannel3     *BackupStream `json:"backup_channel_3"`
        ChannelUniqueID    string        `json:"channelUniqueID"`
}

// M3UChannelStructXEPG : M3U Structure for XEPG
type M3UChannelStructXEPG struct {
        FileM3UID       string `json:"_file.m3u.id,required"`
        FileM3UName     string `json:"_file.m3u.name,required"`
        FileM3UPath     string `json:"_file.m3u.path,required"`
        GroupTitle      string `json:"group-title,required"`
        Name            string `json:"name,required"`
        TvgID           string `json:"tvg-id,required"`
        TvgLogo         string `json:"tvg-logo,required"`
        TvgChno         string `json:"tvg-chno"`
        TvgName         string `json:"tvg-name,required"`
        URL             string `json:"url,required"`
        UUIDKey         string `json:"_uuid.key,required"`
        UUIDValue       string `json:"_uuid.value,required"`
        Values          string `json:"_values,required"`
        LiveEvent       string `json:"liveEvent,required"`
        ChannelUniqueID string `json:"channelUniqueID"`
}

// FilterStruct : Filter Structure
type FilterStruct struct {
        Active         bool   `json:"active"`
        LiveEvent      bool   `json:"liveEvent"`
        CaseSensitive  bool   `json:"caseSensitive"`
        Description    string `json:"description"`
        Exclude        string `json:"exclude"`
        Filter         string `json:"filter"`
        Include        string `json:"include"`
        Name           string `json:"name"`
        Rule           string `json:"rule,omitempty"`
        Type           string `json:"type"`
        StartingNumber string `json:"startingNumber"`
        Category       string `json:"x-category"`
}

// StreamingURLS: Information about all streaming URLs
type StreamingURLS struct {
        Streams map[string]StreamInfo `json:"channels,required"`
}

// StreamInfo: Information about the channel for the streaming URL
type StreamInfo struct {
        ChannelNumber  string        `json:"channelNumber,required"`
        Name           string        `json:"name,required"`
        PlaylistID     string        `json:"playlistID,required"`
        URL            string        `json:"url,required"`
        BackupChannel1 *BackupStream `json:"backup_channel_1,required"`
        BackupChannel2 *BackupStream `json:"backup_channel_2,required"`
        BackupChannel3 *BackupStream `json:"backup_channel_3,required"`
        URLid          string        `json:"urlID,required"`
}

// Notification: Notifications in the web interface
type Notification struct {
        Headline string `json:"headline,required"`
        Message  string `json:"message,required"`
        New      bool   `json:"new,required"`
        Time     string `json:"time,required"`
        Type     string `json:"type,required"`
}

// SettingsStruct: Contents of the settings.json file
type SettingsStruct struct {
        API               bool     `json:"api"`
        AuthenticationAPI bool     `json:"authentication.api"`
        AuthenticationM3U bool     `json:"authentication.m3u"`
        AuthenticationPMS bool     `json:"authentication.pms"`
        AuthenticationWEB bool     `json:"authentication.web"`
        AuthenticationXML bool     `json:"authentication.xml"`
        BackupKeep        int      `json:"backup.keep"`
        BackupPath        string   `json:"backup.path"`
        Branch            string   `json:"git.branch,omitempty"`
        Buffer            string   `json:"buffer"`
        BufferSize        int      `json:"buffer.size.kb"`
        BufferTimeout     float64  `json:"buffer.timeout"`
        CacheImages       bool     `json:"cache.images"`
        EpgSource         string   `json:"epgSource"`
        FFmpegOptions     string   `json:"ffmpeg.options"`
        FFmpegPath        string   `json:"ffmpeg.path"`
        FileM3U           []string `json:"file,omitempty"`  // During the wizard, the M3U is stored in a slice
        FileXMLTV         []string `json:"xmltv,omitempty"` // Old storage system of the provider XML file slice (needed for conversion to the new one)

        Files struct {
                HDHR  map[string]interface{} `json:"hdhr"`
                M3U   map[string]interface{} `json:"m3u"`
                XMLTV map[string]interface{} `json:"xmltv"`
        } `json:"files"`

        FilesUpdate               bool                  `json:"files.update"`
        Filter                    map[int64]interface{} `json:"filter"`
        Key                       string                `json:"key,omitempty"`
        Language                  string                `json:"language"`
        LogEntriesRAM             int                   `json:"log.entries.ram"`
        M3U8AdaptiveBandwidthMBPS int                   `json:"m3u8.adaptive.bandwidth.mbps"`
        MappingFirstChannel       float64               `json:"mapping.first.channel"`
        Port                      string                `json:"port"`
        SSDP                      bool                  `json:"ssdp"`
        TempPath                  string                `json:"temp.path"`
        Tuner                     int                   `json:"tuner"`
        Update                    []string              `json:"update"`
        UpdateURL                 string                `json:"update.url,omitempty"`
        UserAgent                 string                `json:"user.agent"`
        UUID                      string                `json:"uuid"`
        UDPxy                     string                `json:"udpxy"`
        Version                   string                `json:"version"`
        XepgReplaceMissingImages  bool                  `json:"xepg.replace.missing.images"`
        XepgReplaceChannelTitle   bool                  `json:"xepg.replace.channel.title"`
        ThreadfinAutoUpdate       bool                  `json:"ThreadfinAutoUpdate"`
        StoreBufferInRAM          bool                  `json:"storeBufferInRAM"`
        ForceHttps                bool                  `json:"forceHttps"`
        HttpsPort                 int                   `json:"httpsPort"`
        BindIpAddress             string                `json:"bindIpAddress"`
        HttpsThreadfinDomain      string                `json:"httpsThreadfinDomain"`
        HttpThreadfinDomain       string                `json:"httpThreadfinDomain"`
        EnableNonAscii            bool                  `json:"enableNonAscii"`
        EpgCategories             string                `json:"epgCategories"`
        EpgCategoriesColors       string                `json:"epgCategoriesColors"`
        Dummy                     bool                  `json:"dummy"`
        DummyChannel              string                `json:"dummyChannel"`
        IgnoreFilters             bool                  `json:"ignoreFilters"`
}

// LanguageUI: Language for the WebUI
type LanguageUI struct {
        Login struct {
                Failed string
        }
}

type FFProbeOutput struct {
        Streams []ProbeStream `json:"streams"`
}

type ProbeStream struct {
        CodecType     string `json:"codec_type"`
        Width         int    `json:"width,omitempty"`
        Height        int    `json:"height,omitempty"`
        RFrameRate    string `json:"r_frame_rate,omitempty"`
        ChannelLayout string `json:"channel_layout,omitempty"`
        Channels      int    `json:"channels,omitempty"`
}

// Create all system folders
func createSystemFolders() (err error) {

	e := reflect.ValueOf(&System.Folder).Elem()

	for i := 0; i < e.NumField(); i++ {

		var folder = e.Field(i).Interface().(string)

		err = checkFolder(folder)

		if err != nil {
			return
		}

	}

	return
}

// Create all system files
func createSystemFiles() (err error) {
	var debug string
	for _, file := range SystemFiles {

		var filename = getPlatformFile(System.Folder.Config + file)

		err = checkFile(filename)
		if err != nil {
			// File does not exist, will be created now
			err = saveMapToJSONFile(filename, make(map[string]interface{}))
			if err != nil {
				return
			}

			debug = fmt.Sprintf("Create File:%s", filename)
			showDebug(debug, 1)

		}

		switch file {

		case "authentication.json":
			System.File.Authentication = filename
		case "pms.json":
			System.File.PMS = filename
		case "settings.json":
			System.File.Settings = filename
		case "xepg.json":
			System.File.XEPG = filename
		case "urls.json":
			System.File.URLS = filename

		}

	}

	return
}

func updateUrlsJson() {

	getProviderData("m3u", "")
	getProviderData("hdhr", "")

	if Settings.EpgSource == "XEPG" {
		getProviderData("xmltv", "")
	}
	err := buildDatabaseDVR()
	if err != nil {
		ShowError(err, 0)
		return
	}

	buildXEPG(false)
}

// Load settings and set default values (Threadfin)
func loadSettings() (settings SettingsStruct, err error) {

	settingsMap, err := loadJSONFileToMap(System.File.Settings)
	if err != nil {
		return SettingsStruct{}, err
	}

	// Set default values
	var defaults = make(map[string]interface{})
	var dataMap = make(map[string]interface{})

	dataMap["xmltv"] = make(map[string]interface{})
	dataMap["m3u"] = make(map[string]interface{})
	dataMap["hdhr"] = make(map[string]interface{})

	defaults["api"] = false
	defaults["authentication.api"] = false
	defaults["authentication.m3u"] = false
	defaults["authentication.pms"] = false
	defaults["authentication.web"] = false
	defaults["authentication.xml"] = false
	defaults["backup.keep"] = 10
	defaults["backup.path"] = System.Folder.Backup
	defaults["buffer"] = "ffmpeg"
	defaults["buffer.size.kb"] = 1024
	defaults["buffer.timeout"] = 500
	defaults["cache.images"] = false
	defaults["epgSource"] = "XEPG"
	defaults["ffmpeg.options"] = System.FFmpeg.DefaultOptions
	defaults["files"] = dataMap
	defaults["files.update"] = true
	defaults["filter"] = make(map[string]interface{})
	defaults["git.branch"] = System.Branch
	defaults["language"] = "en"
	defaults["log.entries.ram"] = 500
	defaults["mapping.first.channel"] = 1000
	defaults["xepg.replace.missing.images"] = true
	defaults["xepg.replace.channel.title"] = false
	defaults["m3u8.adaptive.bandwidth.mbps"] = 10
	defaults["port"] = "34400"
	defaults["ssdp"] = true
	defaults["storeBufferInRAM"] = true
	defaults["forceHttps"] = false
	defaults["httpsPort"] = 443
	defaults["httpsThreadfinDomain"] = ""
	defaults["httpThreadfinDomain"] = ""
	defaults["enableNonAscii"] = false
	defaults["epgCategories"] = "Kids:kids|News:news|Movie:movie|Series:series|Sports:sports"
	defaults["epgCategoriesColors"] = "kids:mediumpurple|news:tomato|movie:royalblue|series:gold|sports:yellowgreen"
	defaults["tuner"] = 1
	defaults["update"] = []string{"0000"}
	defaults["user.agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"
	defaults["uuid"] = createUUID()
	defaults["udpxy"] = ""
	defaults["version"] = System.DBVersion
	if isRunningInContainer() {
		defaults["ThreadfinAutoUpdate"] = false
	}
	defaults["temp.path"] = System.Folder.Temp

	// Set default values
	for key, value := range defaults {
		if _, ok := settingsMap[key]; !ok {
			settingsMap[key] = value
		}
	}
	err = json.Unmarshal([]byte(mapToJSON(settingsMap)), &settings)
	if err != nil {
		return SettingsStruct{}, err
	}

	// Apply settings from the flags
	if len(System.Flag.Port) > 0 {
		settings.Port = System.Flag.Port
	}

	if len(System.Flag.Branch) > 0 {
		settings.Branch = System.Flag.Branch
		showInfo(fmt.Sprintf("Git Branch:Switching Git Branch to -> %s", settings.Branch))
	}

	if len(settings.FFmpegPath) == 0 {
		settings.FFmpegPath = "/home/threadfin/bin/wrapper"
	}

	// Initialze virutal filesystem for the Buffer
	initBufferVFS()

	settings.Version = System.DBVersion

	err = saveSettings(settings)
	if err != nil {
		return SettingsStruct{}, err
	}

	// Warning if FFmpeg was not found
	if len(Settings.FFmpegPath) == 0 && Settings.Buffer == "ffmpeg" {
		showWarning(2020)
	}

	return settings, nil
}

// Save settings (Threadfin)
func saveSettings(settings SettingsStruct) (err error) {

	if settings.BackupKeep == 0 {
		settings.BackupKeep = 10
	}

	if len(settings.BackupPath) == 0 {
		settings.BackupPath = System.Folder.Backup
	}

	if settings.BufferTimeout < 0 {
		settings.BufferTimeout = 0
	}

	System.Folder.Temp = settings.TempPath + settings.UUID + string(os.PathSeparator)

	err = writeByteToFile(System.File.Settings, []byte(mapToJSON(settings)))
	if err != nil {
		return
	}

	Settings = settings

	setDeviceID()

	return
}

// Enable access via the domain
func setGlobalDomain(domain string) {

	System.Domain = domain

	switch Settings.AuthenticationPMS {
	case true:
		System.Addresses.DVR = "username:password@" + System.Domain
	case false:
		System.Addresses.DVR = System.Domain
	}

	switch Settings.AuthenticationM3U {
	case true:
		System.Addresses.M3U = System.ServerProtocol.M3U + "://" + System.Domain + "/m3u/threadfin.m3u?username=xxx&password=yyy"
	case false:
		System.Addresses.M3U = System.ServerProtocol.M3U + "://" + System.Domain + "/m3u/threadfin.m3u"
	}

	switch Settings.AuthenticationXML {
	case true:
		System.Addresses.XML = System.ServerProtocol.XML + "://" + System.Domain + "/xmltv/threadfin.xml?username=xxx&password=yyy"
	case false:
		System.Addresses.XML = System.ServerProtocol.XML + "://" + System.Domain + "/xmltv/threadfin.xml"
	}

	if Settings.EpgSource != "XEPG" {
		log.Println("SOURCE: ", Settings.EpgSource)
		System.Addresses.M3U = getErrMsg(2106)
		System.Addresses.XML = getErrMsg(2106)
	}

	return
}

// UUID generation
func createUUID() (uuid string) {
	uuid = time.Now().Format("2006-01") + "-" + randomString(4) + "-" + randomString(6)
	return
}

// Generate unique device ID for Plex
func setDeviceID() {

	var id = Settings.UUID

	switch Settings.Tuner {
	case 1:
		System.DeviceID = id

	default:
		System.DeviceID = fmt.Sprintf("%s:%d", id, Settings.Tuner)
	}

	return
}

// Convert provider streaming URL to Threadfin streaming URL
func createStreamingURL(streamingType, playlistID, channelNumber, channelName, url string, backup_channel_1 *BackupStream, backup_channel_2 *BackupStream, backup_channel_3 *BackupStream) (streamingURL string, err error) {

	var streamInfo StreamInfo
	var serverProtocol string

	if len(Data.Cache.StreamingURLS) == 0 {
		Data.Cache.StreamingURLS = make(map[string]StreamInfo)
	}

	var urlID = getMD5(fmt.Sprintf("%s-%s", playlistID, url))

	if s, ok := Data.Cache.StreamingURLS[urlID]; ok {
		streamInfo = s

	} else {
		streamInfo.URL = url
		streamInfo.BackupChannel1 = backup_channel_1
		streamInfo.BackupChannel2 = backup_channel_2
		streamInfo.BackupChannel3 = backup_channel_3
		streamInfo.Name = channelName
		streamInfo.PlaylistID = playlistID
		streamInfo.ChannelNumber = channelNumber
		streamInfo.URLid = urlID

		Data.Cache.StreamingURLS[urlID] = streamInfo

	}

	switch streamingType {

	case "DVR":
		serverProtocol = System.ServerProtocol.DVR

	case "M3U":
		serverProtocol = System.ServerProtocol.M3U

	}

	if Settings.ForceHttps {
		if Settings.HttpsThreadfinDomain != "" {
			serverProtocol = "https"
			System.Domain = Settings.HttpsThreadfinDomain
		}
	}

	streamingURL = fmt.Sprintf("%s://%s/stream/%s", serverProtocol, System.Domain, streamInfo.URLid)
	return
}

func getStreamInfo(urlID string) (streamInfo StreamInfo, err error) {

	if len(Data.Cache.StreamingURLS) == 0 {

		tmp, err := loadJSONFileToMap(System.File.URLS)
		if err != nil {
			return streamInfo, err
		}

		err = json.Unmarshal([]byte(mapToJSON(tmp)), &Data.Cache.StreamingURLS)
		if err != nil {
			return streamInfo, err
		}

	}

	if s, ok := Data.Cache.StreamingURLS[urlID]; ok {
		s.URL = strings.Trim(s.URL, "\r\n")
		s.BackupChannel1 = s.BackupChannel1
		s.BackupChannel2 = s.BackupChannel2
		s.BackupChannel3 = s.BackupChannel3

		streamInfo = s
	} else {
		err = errors.New("streaming error")
	}

	return
}

func isRunningInContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err != nil {
		return false
	}
	return true
}
