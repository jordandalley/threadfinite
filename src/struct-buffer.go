package src

import "time"

// Backup stream structs
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

type StreamInf struct {
	AveragedBandwidth int
	Bandwidth         int
	Framerate         float64
	Resolution        string
	SegmentURL        string
}

type Segment struct {
	Duration     float64
	Info         bool
	PlaylistType string
	Sequence     int64
	URL          string
	Version      int
	Wait         float64
	StreamInf    StreamInf
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

