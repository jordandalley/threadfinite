package src

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/avfs/avfs"
)

// --- System Tools ---

// Checks if the folder exists, if not, the folder is created
func checkFolder(path string) (err error) {

	var debug string
	_, err = os.Stat(filepath.Dir(path))

	if os.IsNotExist(err) {
		// Folder does not exist, will be created now

		err = os.MkdirAll(getPlatformPath(path), 0755)
		if err == nil {

			debug = fmt.Sprintf("Create Folder:%s", path)
			showDebug(debug, 1)

		} else {
			return err
		}

		return nil
	}

	return nil
}

// checkVFSFolder : Checks whether the Folder exists in provided virtual filesystem, if not, the Folder is created
func checkVFSFolder(path string, vfs avfs.VFS) (err error) {

	var debug string
	_, err = vfs.Stat(filepath.Dir(path))

	if fsIsNotExistErr(err) {
		// Folder does not exist, will now be created

		// If we are on Windows and the cache location path is NOT on C:\ we need to create the volume it is located on
		// Failure to do so here will result in a panic error and the stream not playing
		vm := vfs.(avfs.VolumeManager)
		if vfs.OSType() == avfs.OsWindows && avfs.VolumeName(vfs, path) != "C:" {
			vm.VolumeAdd(path)
		}

		err = vfs.MkdirAll(getPlatformPath(path), 0755)
		if err == nil {

			debug = fmt.Sprintf("Create virtual filesystem Folder:%s", path)
			showDebug(debug, 1)

		} else {
			return err
		}

		return nil
	}

	return nil
}

// fsIsNotExistErr : Returns true whether the <err> is known to report that a file or directory does not exist,
// including virtual file system errors
func fsIsNotExistErr(err error) bool {
	if errors.Is(err, fs.ErrNotExist) ||
		errors.Is(err, avfs.ErrWinPathNotFound) ||
		errors.Is(err, avfs.ErrNoSuchFileOrDir) ||
		errors.Is(err, avfs.ErrWinFileNotFound) {
		return true
	}

	return false
}

// Checks if the file exists in the file system
func checkFile(filename string) (err error) {

	var file = getPlatformFile(filename)

	if _, err = os.Stat(file); os.IsNotExist(err) {
		return err
	}

	fi, err := os.Stat(file)
	if err != nil {
		return err
	}

	switch mode := fi.Mode(); {
	case mode.IsDir():
		err = fmt.Errorf("%s: %s", file, getErrMsg(1072))
	case mode.IsRegular():
		break
	}

	return
}

// GetUserHomeDirectory: User home directory
func GetUserHomeDirectory() (userHomeDirectory string) {

	usr, err := user.Current()

	if err != nil {

		for _, name := range []string{"HOME", "USERPROFILE"} {

			if dir := os.Getenv(name); dir != "" {
				userHomeDirectory = dir
				break
			}

		}

	} else {
		userHomeDirectory = usr.HomeDir
	}

	return
}

// Checks file permissions
func checkFilePermission(dir string) (err error) {

	var filename = dir + "permission.test"

	err = os.WriteFile(filename, []byte(""), 0644)
	if err == nil {
		err = os.RemoveAll(filename)
	}

	return
}

// Generate folder path for the current OS
func getPlatformPath(path string) string {
	return filepath.Dir(path) + string(os.PathSeparator)
}

// Generate file path for the current OS
func getPlatformFile(filename string) (osFilePath string) {

	path, file := filepath.Split(filename)
	var newPath = filepath.Dir(path)
	osFilePath = newPath + string(os.PathSeparator) + file

	return
}

// Extract filename from the file path
func getFilenameFromPath(path string) (file string) {
	return filepath.Base(path)
}

// Searches for a file in the OS
func searchFileInOS(file string) (path string) {

	switch runtime.GOOS {

	case "linux", "darwin", "freebsd":
		var args = file
		var cmd = exec.Command("which", strings.Split(args, " ")...)

		out, err := cmd.CombinedOutput()
		if err == nil {

			var slice = strings.Split(strings.Replace(string(out), "\r\n", "\n", -1), "\n")

			if len(slice) > 0 {
				path = strings.Trim(slice[0], "\r\n")
			}

		}

	default:
		return

	}

	return
}

func removeChildItems(dir string) error {

	files, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return err
	}

	for _, file := range files {

		err = os.RemoveAll(file)
		if err != nil {
			return err
		}

	}

	return nil
}

// JSON
func mapToJSON(tmpMap interface{}) string {

	jsonString, err := json.MarshalIndent(tmpMap, "", "  ")
	if err != nil {
		return "{}"
	}

	return string(jsonString)
}

func jsonToMap(content string) map[string]interface{} {

	var tmpMap = make(map[string]interface{})
	json.Unmarshal([]byte(content), &tmpMap)

	return (tmpMap)
}

func jsonToInterface(content string) (tmpMap interface{}, err error) {

	err = json.Unmarshal([]byte(content), &tmpMap)
	return

}

func saveMapToJSONFile(file string, tmpMap interface{}) error {

	var filename = getPlatformFile(file)
	jsonString, err := json.MarshalIndent(tmpMap, "", "  ")

	if err != nil {
		return err
	}

	os.Create(filename)
	err = os.WriteFile(filename, []byte(jsonString), 0644)
	if err != nil {
		return err
	}

	return nil
}

func loadJSONFileToMap(file string) (tmpMap map[string]interface{}, err error) {
	f, err := os.Open(getPlatformFile(file))
	defer f.Close()

	content, err := io.ReadAll(f)

	if err == nil {
		err = json.Unmarshal([]byte(content), &tmpMap)
	}

	f.Close()

	return
}

// Binary
func readByteFromFile(file string) (content []byte, err error) {

	f, err := os.Open(getPlatformFile(file))
	defer f.Close()

	content, err = io.ReadAll(f)
	f.Close()

	return
}

func writeByteToFile(file string, data []byte) (err error) {

	var filename = getPlatformFile(file)
	err = os.WriteFile(filename, data, 0644)

	return
}

func readStringFromFile(file string) (str string, err error) {

	var content []byte
	var filename = getPlatformFile(file)

	err = checkFile(filename)
	if err != nil {
		return
	}

	content, err = os.ReadFile(filename)
	if err != nil {
		ShowError(err, 0)
		return
	}

	str = string(content)

	return
}

// Network
func resolveHostIP() error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return err
		}

		for _, addr := range addrs {
			networkIP, ok := addr.(*net.IPNet)
			System.IPAddressesList = append(System.IPAddressesList, networkIP.IP.String())

			if ok {
				ip := networkIP.IP.String()

				if networkIP.IP.To4() != nil {
					// Skip unwanted IPs
					if !strings.HasPrefix(ip, "169.254") {
						System.IPAddressesV4 = append(System.IPAddressesV4, ip)
						System.IPAddress = ip
					}
				} else {
					System.IPAddressesV6 = append(System.IPAddressesV6, ip)
				}
			}
		}
	}

	if len(System.IPAddress) == 0 {
		if len(System.IPAddressesV4) > 0 {
			System.IPAddress = System.IPAddressesV4[0]
		} else if len(System.IPAddressesV6) > 0 {
			System.IPAddress = System.IPAddressesV6[0]
		}
	}

	System.Hostname, err = os.Hostname()
	if err != nil {
		return err
	}

	return nil
}

// Miscellaneous
func randomString(n int) string {

	const alphanum = "AB1CD2EF3GH4IJ5KL6MN7OP8QR9ST0UVWXYZ"

	var bytes = make([]byte, n)

	rand.Read(bytes)

	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}

	return string(bytes)
}

func parseTemplate(content string, tmpMap map[string]interface{}) (result string) {

	t := template.Must(template.New("template").Parse(content))

	var tpl bytes.Buffer

	if err := t.Execute(&tpl, tmpMap); err != nil {
		ShowError(err, 0)
	}
	result = tpl.String()

	return
}

func indexOfString(element string, data []string) int {

	for k, v := range data {
		if element == v {
			return k
		}
	}

	return -1
}

func indexOfFloat64(element float64, data []float64) int {

	for k, v := range data {
		if element == v {
			return (k)
		}
	}

	return -1
}

func getMD5(str string) string {

	md5Hasher := md5.New()
	md5Hasher.Write([]byte(str))

	return hex.EncodeToString(md5Hasher.Sum(nil))
}

func getBaseUrl(host string, port string) string {
	if strings.Contains(host, ":") {
		return host
	} else {
		return fmt.Sprintf("%s:%s", host, port)
	}
}
