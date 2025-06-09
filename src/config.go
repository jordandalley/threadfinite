package src

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"github.com/avfs/avfs"
)

// System : Contains all system information
var System SystemStruct

// WebScreenLog : Logs are stored in RAM and made available for the web interface
var WebScreenLog WebScreenLogStruct

// Settings : Content of settings.json
var Settings SettingsStruct

// Data : All data is stored here (Lineup, XMLTV)
var Data DataStruct

// SystemFiles : All system files
var SystemFiles = []string{"authentication.json", "pms.json", "settings.json", "xepg.json", "urls.json"}

// BufferInformation : Information about the buffer (active streams, maximum streams)
var BufferInformation sync.Map

// bufferVFS : Filesystem to use for the buffer
var bufferVFS avfs.VFS

// BufferClients : Number of clients playing a stream via the buffer
var BufferClients sync.Map

// Lock : Lock Map
var Lock = sync.RWMutex{}

var (
	xepgMutex   sync.Mutex
	infoMutex   sync.Mutex
	logMutex    sync.Mutex
	systemMutex sync.Mutex
)

// Init : System initialisation
func Init() (err error) {

	var debug string
        // System settings
	System.AppName = strings.ToLower(System.Name)
	System.ARCH = runtime.GOARCH
	System.OS = runtime.GOOS
	System.ServerProtocol.API = "http"
	System.ServerProtocol.DVR = "http"
	System.ServerProtocol.M3U = "http"
	System.ServerProtocol.WEB = "http"
	System.ServerProtocol.XML = "http"
	System.PlexChannelLimit = 480
	System.UnfilteredChannelLimit = 480
	System.Compatibility = "0.1.0"

        // FFmpeg default settings
	System.FFmpeg.DefaultOptions = "-i [URL]"

        // Default log entries, later overridden by those in settings.json.
        // Needed so the first entries are displayed in the log (webUI).
	Settings.LogEntriesRAM = 500

        // Define folder paths
	var tempFolder = os.TempDir() + string(os.PathSeparator) + System.AppName + string(os.PathSeparator)
	tempFolder = getPlatformPath(strings.Replace(tempFolder, "//", "/", -1))

	if len(System.Folder.Config) == 0 {
		System.Folder.Config = GetUserHomeDirectory() + string(os.PathSeparator) + "." + System.AppName + string(os.PathSeparator)
	} else {
		System.Folder.Config = strings.TrimRight(System.Folder.Config, string(os.PathSeparator)) + string(os.PathSeparator)
	}

	System.Folder.Config = getPlatformPath(System.Folder.Config)
	System.Folder.Backup = System.Folder.Config + "backup" + string(os.PathSeparator)
	System.Folder.Data = System.Folder.Config + "data" + string(os.PathSeparator)
	System.Folder.Cache = System.Folder.Config + "cache" + string(os.PathSeparator)
	System.Folder.ImagesCache = System.Folder.Cache + "images" + string(os.PathSeparator)
	System.Folder.ImagesUpload = System.Folder.Data + "images" + string(os.PathSeparator)
	System.Folder.Temp = tempFolder

        // Create system folders
	err = createSystemFolders()
	if err != nil {
		ShowError(err, 1070)
		return
	}

	if len(System.Flag.Restore) > 0 {
                // Settings are restored via CLI. Further initialization is not necessary.
		return
	}

	System.File.XML = getPlatformFile(fmt.Sprintf("%s%s.xml", System.Folder.Data, System.AppName))
	System.File.M3U = getPlatformFile(fmt.Sprintf("%s%s.m3u", System.Folder.Data, System.AppName))

	System.Compressed.GZxml = getPlatformFile(fmt.Sprintf("%s%s.xml.gz", System.Folder.Data, System.AppName))

	err = activatedSystemAuthentication()
	if err != nil {
		return
	}

	err = resolveHostIP()
	if err != nil {
		ShowError(err, 1002)
	}

        // Menu for the web interface
	System.WEB.Menu = []string{"playlist", "xmltv", "filter", "mapping", "users", "settings", "log", "logout"}

	fmt.Println("For help run: " + getPlatformFile(os.Args[0]) + " -h")
	fmt.Println()

	if System.Flag.Debug > 0 {
		debug = fmt.Sprintf("Debug Level:%d", System.Flag.Debug)
		showDebug(debug, 1)
	}

	showInfo(fmt.Sprintf("Version:%s Build: %s", System.Version, System.Build))
	showInfo(fmt.Sprintf("Database Version:%s", System.DBVersion))
	showInfo(fmt.Sprintf("System IP Addresses:IPv4: %d | IPv6: %d", len(System.IPAddressesV4), len(System.IPAddressesV6)))
	showInfo("Hostname:" + System.Hostname)
	showInfo(fmt.Sprintf("System Folder:%s", getPlatformPath(System.Folder.Config)))

        // Create system files (if not present)
	err = createSystemFiles()
	if err != nil {
		ShowError(err, 1071)
		return
	}

        // Load settings (settings.json)
	showInfo(fmt.Sprintf("Load Settings:%s", System.File.Settings))

	_, err = loadSettings()
	if err != nil {
		ShowError(err, 0)
		return
	}

        // Check permissions of all folders
	err = checkFilePermission(System.Folder.Config)
	if err == nil {
		err = checkFilePermission(System.Folder.Temp)
	}

	showInfo(fmt.Sprintf("Temporary Folder:%s", getPlatformPath(System.Folder.Temp)))

	err = checkFolder(System.Folder.Temp)
	if err != nil {
		return
	}

	err = removeChildItems(getPlatformPath(System.Folder.Temp))
	if err != nil {
		return
	}

	// Set base URI
	if Settings.HttpThreadfinDomain != "" {
		setGlobalDomain(getBaseUrl(Settings.HttpThreadfinDomain, Settings.Port))
	} else {
		setGlobalDomain(fmt.Sprintf("%s:%s", System.IPAddress, Settings.Port))
	}

	System.URLBase = fmt.Sprintf("%s://%s:%s", System.ServerProtocol.WEB, System.IPAddress, Settings.Port)

        // Start DLNA server
	if Settings.SSDP {
		err = SSDP()
		if err != nil {
			return
		}
	}

        // Load HTML files
	loadHTMLMap()

	return
}

// StartSystem : Starts the system
func StartSystem(updateProviderFiles bool) (err error) {

	setDeviceID()

	if System.ScanInProgress == 1 {
		return
	}

	// System information for the console output
	showInfo(fmt.Sprintf("UUID:%s", Settings.UUID))
	showInfo(fmt.Sprintf("Tuner (Jellyfin / Plex / Emby):%d", Settings.Tuner))
	showInfo(fmt.Sprintf("EPG Source:%s", Settings.EpgSource))
	showInfo(fmt.Sprintf("Plex Channel Limit:%d", System.PlexChannelLimit))
	showInfo(fmt.Sprintf("Unfiltered Chan. Limit:%d", System.UnfilteredChannelLimit))

        // Update provider data
	if len(Settings.Files.M3U) > 0 && Settings.FilesUpdate == true || updateProviderFiles == true {

		err = ThreadfinAutoBackup()
		if err != nil {
			ShowError(err, 1090)
		}

		getProviderData("m3u", "")
		getProviderData("hdhr", "")

		if Settings.EpgSource == "XEPG" {
			getProviderData("xmltv", "")
		}

	}

	err = buildDatabaseDVR()
	if err != nil {
		ShowError(err, 0)
		return
	}

	buildXEPG(true)

	return
}
