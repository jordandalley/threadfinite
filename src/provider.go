package src

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	m3u "threadfin/src/internal/m3u-parser"
)

// fileType: Which file type should be updated (m3u, hdhr, xml) | fileID: Update a specific file (provider ID)
func getProviderData(fileType, fileID string) (err error) {

	var fileExtension, serverFileName string
	var body = make([]byte, 0)
	var newProvider = false
	var dataMap = make(map[string]interface{})

	var saveDateFromProvider = func(fileSource, serverFileName, id string, body []byte) (err error) {

		var data = make(map[string]interface{})

		if value, ok := dataMap[id].(map[string]interface{}); ok {
			data = value
		} else {
			data["id.provider"] = id
			dataMap[id] = data
		}

                // Default keys for the provider data
		var keys = []string{"name", "description", "type", "file." + System.AppName, "file.source", "tuner", "http_proxy.ip", "http_proxy.port", "last.update", "compatibility", "counter.error", "counter.download", "provider.availability"}

		for _, key := range keys {

			if _, ok := data[key]; !ok {

				switch key {

				case "name":
					data[key] = serverFileName

				case "description":
					data[key] = ""

				case "type":
					data[key] = fileType

				case "file." + System.AppName:
					data[key] = id + fileExtension

				case "file.source":
					data[key] = fileSource

				case "http_proxy.ip":
					data[key] = ""

				case "http_proxy.port":
					data[key] = ""

				case "last.update":
					data[key] = time.Now().Format("2006-01-02 15:04:05")

				case "tuner":
					if fileType == "m3u" || fileType == "hdhr" {
						if _, ok := data[key].(float64); !ok {
							data[key] = 1
						}
					}

				case "compatibility":
					data[key] = make(map[string]interface{})

				case "counter.download":
					data[key] = 0.0

				case "counter.error":
					data[key] = 0.0

				case "provider.availability":
					data[key] = 100
				}

			}

		}

		if _, ok := data["id.provider"]; !ok {
			data["id.provider"] = id
		}

		// Extract file
		body, err = extractGZIP(body, fileSource)
		if err != nil {
			ShowError(err, 000)
			return
		}

		// Verify data
		showInfo("Check File:" + fileSource)

		switch fileType {

		case "m3u":
			newM3u, err := m3u.MakeInterfaceFromM3U(body)
			if err != nil {
				return err
			}

			var m3uContent strings.Builder
			m3uContent.WriteString("#EXTM3U\n")

			for _, channel := range newM3u {
				channelMap := channel.(map[string]string)

				extinf := fmt.Sprintf(`#EXTINF:-1 tvg-id="%s" tvg-name="%s" tvg-chno="%s" tvg-logo="%s" group-title="%s",%s`,
					channelMap["tvg-id"],
					channelMap["tvg-name"],
					channelMap["tvg-chno"],
					channelMap["tvg-logo"],
					channelMap["group-title"],
					channelMap["name"],
				)

				m3uContent.WriteString(extinf + "\n" + channelMap["url"] + "\n")
			}

			m3uBytes := []byte(m3uContent.String())
			body = m3uBytes

		case "hdhr":
			_, err = jsonToInterface(string(body))

		case "xmltv":
			err = checkXMLCompatibility(id, body)

		}

		if err != nil {
			return
		}

		var filePath = System.Folder.Data + data["file."+System.AppName].(string)

		err = writeByteToFile(filePath, body)

		if err == nil {
			data["last.update"] = time.Now().Format("2006-01-02 15:04:05")
			data["counter.download"] = data["counter.download"].(float64) + 1
		}

		return

	}

	switch fileType {

	case "m3u":
		dataMap = Settings.Files.M3U
		fileExtension = ".m3u"

	case "hdhr":
		dataMap = Settings.Files.HDHR
		fileExtension = ".json"

	case "xmltv":
		dataMap = Settings.Files.XMLTV
		fileExtension = ".xml"

	}

	for dataID, d := range dataMap {

		var data = d.(map[string]interface{})
		var fileSource = data["file.source"].(string)
		var httpProxyIp = ""
		if data["http_proxy.ip"] != nil {
			httpProxyIp = data["http_proxy.ip"].(string)
		}
		var httpProxyPort = ""
		if data["http_proxy.port"] != nil {
			httpProxyPort = data["http_proxy.port"].(string)
		}
		var httpProxyUrl = ""
		if httpProxyIp != "" && httpProxyPort != "" {
			httpProxyUrl = fmt.Sprintf("http://%s:%s", httpProxyIp, httpProxyPort)
		}

		newProvider = false

		if _, ok := data["new"]; ok {
			newProvider = true
			delete(data, "new")
		}

                // If an ID is present and does not match the one in the database, the update is skipped (goto)
		if len(fileID) > 0 && newProvider == false {
			if dataID != fileID {
				goto Done
			}
		}

		switch fileType {

		case "hdhr":

                        // Loading from HDHomeRun tuner
			showInfo("Tuner:" + fileSource)
			var tunerURL = "http://" + fileSource + "/lineup.json"
			serverFileName, body, err = downloadFileFromServer(tunerURL, httpProxyUrl)

		default:

			if strings.Contains(fileSource, "http://") || strings.Contains(fileSource, "https://") {

				// Loading from Remote Server
				showInfo("Download:" + fileSource)
				serverFileName, body, err = downloadFileFromServer(fileSource, httpProxyUrl)

			} else {

                                // Loading a local file
				showInfo("Open:" + fileSource)

				err = checkFile(fileSource)
				if err == nil {
					body, err = readByteFromFile(fileSource)
					serverFileName = getFilenameFromPath(fileSource)
				}

			}

		}

		if err == nil {

			err = saveDateFromProvider(fileSource, serverFileName, dataID, body)
			if err == nil {
				showInfo("Save File:" + fileSource + " [ID: " + dataID + "]")
			}

		}

		if err != nil {

			ShowError(err, 000)
			var downloadErr = err

			if newProvider == false {

                                // Check if older file exists
				var file = System.Folder.Data + dataID + fileExtension

				err = checkFile(file)
				if err == nil {

					if len(fileID) == 0 {
						showWarning(1011)
					}

					err = downloadErr
				}

                                // Increment error counter by 1
				var data = make(map[string]interface{})
				if value, ok := dataMap[dataID].(map[string]interface{}); ok {

					data = value
					data["counter.error"] = data["counter.error"].(float64) + 1
					data["counter.download"] = data["counter.download"].(float64) + 1

				}

			} else {
				return downloadErr
			}

		}

		// Calculate error rate
		if newProvider == false {

			if value, ok := dataMap[dataID].(map[string]interface{}); ok {

				var data = make(map[string]interface{})
				data = value

				if data["counter.error"].(float64) == 0 {
					data["provider.availability"] = 100
				} else {
					data["provider.availability"] = int(data["counter.error"].(float64)*100/data["counter.download"].(float64)*-1 + 100)
				}

			}

		}

		switch fileType {

		case "m3u":
			Settings.Files.M3U = dataMap

		case "hdhr":
			Settings.Files.HDHR = dataMap

		case "xmltv":
			Settings.Files.XMLTV = dataMap
			delete(Data.Cache.XMLTV, System.Folder.Data+dataID+fileExtension)

		}

		saveSettings(Settings)

	Done:
	}

	return
}

func downloadFileFromServer(providerURL string, proxyUrl string) (filename string, body []byte, err error) {
	_, err = url.ParseRequestURI(providerURL)
	if err != nil {
		return
	}

	httpClient := &http.Client{}

	if proxyUrl != "" {
		proxyURL, err := url.Parse(proxyUrl)
		if err != nil {
			return "", nil, err
		}

		httpClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}
	}

	req, err := http.NewRequest("GET", providerURL, nil)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", Settings.UserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	resp.Header.Set("User-Agent", Settings.UserAgent)

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("%d: %s %s", resp.StatusCode, providerURL, http.StatusText(resp.StatusCode))
		return
	}

	// Get filename from the header
	var index = strings.Index(resp.Header.Get("Content-Disposition"), "filename")

	if index > -1 {
		var headerFilename = resp.Header.Get("Content-Disposition")[index:]
		var value = strings.Split(headerFilename, `=`)
		var f = strings.Replace(value[1], `"`, "", -1)
		f = strings.Replace(f, `;`, "", -1)
		filename = f
		showInfo("Header filename:" + filename)
	} else {
		var cleanFilename = strings.SplitN(getFilenameFromPath(providerURL), "?", 2)
		filename = cleanFilename[0]
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	return
}
