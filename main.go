package main

import (
	"crypto/md5"
	"errors" // standard console
	"io"
	"path/filepath"
	"runtime"

	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	//"github.com/fsnotify/fsnotify"
)

// log
// some tests on filenames
// files
// list files in a directory
// comparison of two arrayw of bytes
// Get the parent directory of a file

//import "encoding/hex"
// to be notified when a repertory is modified (file creation ,deletion etc)

var logFileName = "rocksmithprofilesync.log"
var syncDirectory = "..\\Sync"
var storageDirectory = "."
var filesHashMap = make(map[string]([]byte)) // Map referenced by the file names which keeps the MD5 hash of all files (profiles and localprofiles.json)
var filesLastAccessMap = make(map[string]time.Time)
var minPeriodBetweenFileEvents int64 = 700000000 // nanoseconds
var localProfileID []byte

// Open a files, and keep trying a few times if this file is already used by another process
func insistentOpen(filePath string) (*os.File, error) {
	var numberOfTries = 60
	file, err := os.Open(filePath)
	pathErr, ok := err.(*os.PathError)
	for numberOfTries > 0 && ok && pathErr != os.ErrExist && pathErr != os.ErrInvalid && pathErr != os.ErrNotExist && pathErr != os.ErrPermission && err != nil {
		file, err = os.Open(filePath)
		pathErr, ok = err.(*os.PathError)
		numberOfTries--
		// Sometimes the file can't be accessed because of syncthing-inotify locking the file
		log.Printf("Erreur : le fichier est inaccessible. Nouvel essai (%d restant)...\n", numberOfTries)
		time.Sleep(time.Second)
	}

	//log.Println("file :", file)
	return file, err
}

func isProfileDataBaseFile(fileName string) bool {
	var result = false
	if strings.HasSuffix(fileName, "_prfldb") {
		result = true
	} else {
		result = false
	}
	return result
}

func isLocalProfilesJSONFile(fileName string) bool {
	var result = false
	if fileName == "localprofiles.json" {
		result = true
	} else {
		result = false
	}
	return result
}

func saveEventTime(fileName string) {
	var eventTime = time.Now()
	//eventTime = eventTime.Truncate(minPeriodBetweenFileEvents)
	filesLastAccessMap[fileName] = eventTime
}

func eventIsTooRecent(fileName string) bool {
	lastEventTime := filesLastAccessMap[fileName]
	now := time.Now()
	sinceLastEvent := now.Sub(lastEventTime)
	var sinceLastEventNanoSec = sinceLastEvent.Nanoseconds()
	var result = false
	//log.Println("sinceLastEventNanoSec :", sinceLastEventNanoSec)
	//log.Println("minPeriodBetweenFileEvents :", minPeriodBetweenFileEvents)
	if sinceLastEventNanoSec <= minPeriodBetweenFileEvents {
		//log.Println("Too recent")
		result = true
	} else {
		//log.Println("Ok")
		result = false
	}
	return result
}

func saveHash(fileName string, parentFolder string) error {
	var result []byte
	file, err := insistentOpen(parentFolder + string(filepath.Separator) + fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	filesHashMap[fileName] = hash.Sum(result)
	return nil
}

func hasSameHash(fileName string, parentFolder string) (bool, error) {
	file, err := insistentOpen(parentFolder + string(filepath.Separator) + fileName)
	if err != nil {
		return false, err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false, err
	}
	var result []byte
	fileHash := hash.Sum(result)
	var boolresult = false
	if bytes.Equal(fileHash, filesHashMap[fileName]) {
		boolresult = true
	} else {
		boolresult = false
	}
	return boolresult, nil
}

func deleteHash(fileName string) {
	delete(filesHashMap, fileName)
}

func importFile(fileName string, profileID []byte) error {
	if len(profileID) != 4 {
		err := errors.New("Longueur du profile ID incorrecte !")
		return err
	}
	srcFile, err := insistentOpen(syncDirectory + string(filepath.Separator) + fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(storageDirectory + string(filepath.Separator) + fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer dstFile.Close()

	// Copy the first 8 bytes
	var nbCopiedBytes int64 = 8
	var buffer = make([]byte, 8)
	nbReadBytes, err := srcFile.Read(buffer)
	if err != nil || nbReadBytes != len(buffer) {
		log.Fatal(err)
	}
	nbWrittenBytes, err := dstFile.Write(buffer)
	if err != nil || nbWrittenBytes != len(buffer) {
		log.Fatal(err)
	}

	// Ignore the 4-bytes long profile ID we don't need
	var profileIDLength = 4
	nbReadBytes, err = srcFile.Read(make([]byte, profileIDLength))
	if nbReadBytes != profileIDLength {
		err := errors.New("Incorrect profile ID length !")
		return err
	}
	if err != nil {
		log.Fatal(err)
	}

	// Copy the profile ID which belongs to this PC
	nbWrittenBytes, err = dstFile.Write(profileID)
	if nbWrittenBytes != 4 || err != nil {
		log.Fatal(err)
	}
	nbCopiedBytes = nbCopiedBytes + 4

	// Copy the rest of the file...
	//srcReader := bufio.NewReader(srcFile)
	//dstWriter := bufio.NewWriter(dstFile)
	n, err := io.Copy(dstFile, srcFile)
	nbCopiedBytes += n
	if err != nil {
		log.Fatal(err)
	}
	//log.Printf("%d copied bytes\n", nbCopiedBytes+12) // To delete after debugging

	//dstFile.Write(b []byte)

	//nbReadBytes, err = srcFile.ReadAt(profileID, 8)
	//dstFile.Write(b []byte)
	return nil
}

func exportFile(fileName string) error {

	srcFile, err := insistentOpen(storageDirectory + string(filepath.Separator) + fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(syncDirectory + string(filepath.Separator) + fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		log.Fatal(err)
	}
	//log.Println("File exported and now ready to be synchronized : ", fileName)
	return nil
}

func hasProfileID(fileName string, parentFolder string, wantedProfileID []byte) (bool, error) {
	if len(wantedProfileID) != 4 {
		err := errors.New("Incorrect profile ID length !")
		return false, err
	}
	testedFile, err := insistentOpen(storageDirectory + string(filepath.Separator) + fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer testedFile.Close()
	var testedProfileID = make([]byte, 4)
	testedFile.ReadAt(testedProfileID, 8)
	var result = false
	if bytes.Equal(testedProfileID, wantedProfileID) {
		result = true
	} else {
		result = false
	}
	return result, nil
}

func inSyncFolder(filePath string) bool {
	parentFolder := filepath.Dir(filePath)
	var result = false
	if path.Clean(parentFolder) == path.Clean(syncDirectory) {
		result = true
	} else {
		result = false
	}
	return result
}

func inStorageFolder(filePath string) bool {
	parentFolder := filepath.Dir(filePath)
	var result = false
	if path.Clean(parentFolder) == path.Clean(storageDirectory) {
		result = true
	} else {
		result = false
	}
	return result
}

func filesManager() {
	filesWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer filesWatcher.Close()
	err = filesWatcher.Add(syncDirectory)
	if err != nil {
		log.Fatal(err)
	}
	err = filesWatcher.Add(storageDirectory)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case event := <-filesWatcher.Events:

			//log.Println("Évènement :", event)
			// Do something only if the file is a profile file or localprofiles.json
			if isProfileDataBaseFile(filepath.Base(event.Name)) || isLocalProfilesJSONFile(filepath.Base(event.Name)) {
				if eventIsTooRecent(filepath.Base(event.Name)) {
				} else {
					//log.Println("Event : ", event)
					if (event.Op&fsnotify.Write == fsnotify.Write) || event.Op&fsnotify.Create == fsnotify.Create {

						if inSyncFolder(event.Name) {
							hasProfileID, _ := hasProfileID(event.Name, syncDirectory, localProfileID)
							if hasProfileID {
								// Don't do anything
							} else {
								importFile(filepath.Base(event.Name), localProfileID)
								err = saveHash(filepath.Base(event.Name), syncDirectory)
								if err != nil {
									log.Fatal(err)
								}
								if event.Op&fsnotify.Create == fsnotify.Create {
									if isProfileDataBaseFile(filepath.Base(event.Name)) {
										log.Println("New profile created by another PC !", filepath.Base(event.Name))
									} else if isLocalProfilesJSONFile(filepath.Base(event.Name)) {
										log.Println("localprofiles.json file created by another PC")
									}
								} else {
									if isProfileDataBaseFile(filepath.Base(event.Name)) {
										log.Println("Profile has been updated from another PC: ", filepath.Base(event.Name))
									} else if isLocalProfilesJSONFile(filepath.Base(event.Name)) {
										log.Println("localprofiles.json has been updated from another PC")
									}
								}
							}
						} else if inStorageFolder(event.Name) {
							hasSameHash, _ := hasSameHash(filepath.Base(event.Name), storageDirectory)
							if hasSameHash {
								// Don't do anything
							} else {
								exportFile(filepath.Base(event.Name))

								if event.Op&fsnotify.Create == fsnotify.Create {
									if isProfileDataBaseFile(filepath.Base(event.Name)) {
										log.Println("New profile created by Rocksmith!", filepath.Base(event.Name))
									} else if isLocalProfilesJSONFile(filepath.Base(event.Name)) {
										log.Println("localprofiles.json created by Rocksmith")
									}
								} else {
									if isProfileDataBaseFile(filepath.Base(event.Name)) {
										log.Println("Profil has been updated by Rocksmith: ", filepath.Base(event.Name))
									} else if isLocalProfilesJSONFile(filepath.Base(event.Name)) {
										log.Println("localprofiles.json has been updated by Rocksmith")
									}

								}
							}
						}

						// A file is deleted
					} else if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename {
						if inSyncFolder(event.Name) {

							oldPath := storageDirectory + string(filepath.Separator) + filepath.Base(event.Name)
							newPath := oldPath + ".old"
							os.Rename(oldPath, newPath)
							deleteHash(filepath.Base(event.Name))
							log.Println("File has been deleted by another PC (added \".old\" extension): ", filepath.Base(event.Name))
						} else if inStorageFolder(event.Name) {
							oldPath := syncDirectory + string(filepath.Separator) + filepath.Base(event.Name)
							newPath := oldPath + ".old"
							os.Rename(oldPath, newPath)
							deleteHash(filepath.Base(event.Name))
							if isProfileDataBaseFile(filepath.Base(event.Name)) {
								log.Println("Profile has been deleted by Rocksmith (added \".old\" extension)! ", filepath.Base(event.Name))
							} else if isLocalProfilesJSONFile(filepath.Base(event.Name)) {
								log.Println("localprofiles.json has been deleted by Rocksmith (added \".old\" extension)! ", filepath.Base(event.Name))
							}

						}
					}
					saveEventTime(filepath.Base(event.Name))
				}
			}
		case err := <-filesWatcher.Errors:
			log.Println("Error: ", err)
		}
	}
}

func main() {
	logFile, err := os.Create(logFileName)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	localprofiles, err := insistentOpen(storageDirectory + string(filepath.Separator) + "localprofiles.json")
	if err != nil {
		log.Println("Make sure you put this executable in the \"Storage\" folder which contains the Rocksmith profiles.")
		log.Fatal(err)
	}
	// Creates a profile ID variable with a length of 4 bytes, to get the profile ID from the localprofiles.json file
	profileID := make([]byte, 4)
	nbReadBytes, err := localprofiles.ReadAt(profileID, 8)
	if nbReadBytes != 4 || err != nil {
		log.Fatal("Problem with localprofiles.json\n", err)
	}
	localProfileID = profileID
	log.Println("Analysing localprofiles.json...")
	log.Printf("Profile ID found: 0x%08x\n", localProfileID)
	err = localprofiles.Close()
	if err != nil {
		log.Fatal(err)
	}

	// Make a list of all files to be imported from Sync to Storage
	syncFiles, _ := ioutil.ReadDir(syncDirectory)
	log.Println("List of files to retrieve from the sync folder:")
	var syncFilesToImport []os.FileInfo
	for _, file := range syncFiles {
		if isProfileDataBaseFile(file.Name()) || isLocalProfilesJSONFile(file.Name()) {
			// Add these files to the list of files that must be imported
			syncFilesToImport = append(syncFilesToImport, file)
		}
	}
	for _, file := range syncFilesToImport {
		log.Println(file.Name())
	}
	log.Println("Updating local profiles with the data from synchronized profiles:")
	// Delete old profiles in the Storage folder (adding a ".old" to their name)
	storageFiles, _ := ioutil.ReadDir(storageDirectory)
	//log.Println("List of deleted files in the storage folder (added .old extension):")
	for _, file := range storageFiles {
		if strings.HasSuffix(file.Name(), "_prfldb") || file.Name() == "localprofiles.json" {
			log.Println("Deleted file (.old):", file.Name())
			err = os.Rename(file.Name(), file.Name()+".old")
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	// Import the files from Sync to Storage
	for _, file := range syncFilesToImport {
		err := importFile(file.Name(), localProfileID)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Imported file: ", file.Name())
	}
	log.Println("Ready! make sure the ", syncDirectory, " folder is synchronized. Syncthing must be running.")

	go filesManager()

	for {
		time.Sleep(time.Hour)
		runtime.GC()
	}
}
