package src

import (
	"fmt"
	"math/rand"
	"time"
)

// InitMaintenance : Initialize maintenance process
func InitMaintenance() (err error) {

	rand.Seed(time.Now().Unix())
	System.TimeForAutoUpdate = fmt.Sprintf("0%d%d", randomTime(0, 2), randomTime(10, 59))

	go maintenance()

	return
}

func maintenance() {

	for {

		var t = time.Now()

                // Update playlist and XMLTV files
		systemMutex.Lock()
		if System.ScanInProgress == 0 {
			systemMutex.Unlock()
			for _, schedule := range Settings.Update {

				if schedule == t.Format("1504") {

					showInfo("Update:" + schedule)

					// Create backup
					err := ThreadfinAutoBackup()
					if err != nil {
						ShowError(err, 000)
					}

					// Update playlist and XMLTV files
					getProviderData("m3u", "")
					getProviderData("hdhr", "")

					if Settings.EpgSource == "XEPG" {
						getProviderData("xmltv", "")
					}

					// Create database for DVR
					err = buildDatabaseDVR()
					if err != nil {
						ShowError(err, 000)
					}

					systemMutex.Lock()
					if !Settings.CacheImages && System.ImageCachingInProgress == 0 {
						systemMutex.Unlock()
						removeChildItems(System.Folder.ImagesCache)
					} else {
						systemMutex.Unlock()
					}

					// Create XEPG files
					systemMutex.Lock()
					Data.Cache.XMLTV = make(map[string]XMLTV)
					systemMutex.Unlock()

					buildXEPG(false)

				}

			}

		} else {
			systemMutex.Unlock()
		}

		time.Sleep(60 * time.Second)

	}

	return
}

func randomTime(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}
