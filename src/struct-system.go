package src

import "threadfin/src/internal/imgcache"

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
