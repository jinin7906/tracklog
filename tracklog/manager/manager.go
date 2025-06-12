package manager

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"tracklog/config"
)

type LogLine struct {
	MonitorName string
	Content     string
	Timestamp   time.Time
}

type Mgr struct {
	IsRun bool
}

func NewMgr() *Mgr {
	var mng *Mgr = new(Mgr)
	mng.IsRun = true

	return mng
}

func (This *Mgr) Start(monCfgs *[]config.MonitorConfig, wg *sync.WaitGroup) bool {

	go This.compressFile(monCfgs)
	wg.Add(1)

	return true
}

func (This *Mgr) SendLogToTCP(address, data string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("[tcp_client] connect err %s: %v", address, err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(data + "\n"))
	if err != nil {
		return fmt.Errorf("[tcp_client] data send err %s: %v", address, err)
	}
	return nil
}

func (This *Mgr) GetMatchingFiles(directoryPath, pattern string) ([]string, error) {
	fullPattern := filepath.Join(directoryPath, pattern)

	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, fmt.Errorf("GetMatchingFiles Read Error[pattern: %s]: %w", fullPattern, err)
	}

	var filenames []string
	for _, match := range matches {
		filenames = append(filenames, filepath.Base(match))
	}

	//fmt.Printf("file list. '%s'\n", filenames)
	return filenames, nil
}

func (This *Mgr) AppendToFile(originalFilePath, content string) error {

	currentTime := time.Now()
	baseDir := filepath.Dir(originalFilePath)
	fileNameWithExt := filepath.Base(originalFilePath)

	year := currentTime.Format("2006")
	month := currentTime.Format("01")

	finalDir := filepath.Join(baseDir, year, month)

	dirReady, err := This.CreateDir(finalDir)
	if !dirReady {
		return fmt.Errorf("failed to create directory for file %s: %w", finalDir, err)
	}

	timeSuffix := currentTime.Format("_02")

	dotIndex := strings.LastIndex(fileNameWithExt, ".")
	var newFileName string
	if dotIndex == -1 {
		newFileName = fmt.Sprintf("%s%s", fileNameWithExt, timeSuffix)
	} else {
		name := fileNameWithExt[:dotIndex]
		ext := fileNameWithExt[dotIndex:]
		newFileName = fmt.Sprintf("%s%s%s", name, timeSuffix, ext)
	}

	finalFilePath := filepath.Join(finalDir, newFileName)

	fmt.Printf("[FileName] : %s | [content] : %s\n", finalFilePath, content)

	//f, err := os.OpenFile(finalFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	f, err := os.OpenFile(finalFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("tracklog file open err: %s: %w", finalFilePath, err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("tracklog file write err: %s: %w", finalFilePath, err)
	}

	return nil
}

// create dir
func (This *Mgr) CreateDir(dirPath string) (bool, error) {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		fmt.Printf("Attempting to create directory: %s\n", dirPath)
		err = os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			return false, fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}
		fmt.Printf("Created directory: %s\n", dirPath)
		return true, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check directory %s: %w", dirPath, err)
	} else {
		//fmt.Printf("Directory already exists: %s\n", dirPath)
		return true, nil
	}
}

//######################################################################### private

// 압축 goroutine
func (This *Mgr) compressFile(monCfgs *[]config.MonitorConfig) {
	fmt.Println("start file compression task. it will run once every day.")

	for {
		fmt.Printf("--- start file compression task: %s ---\n", time.Now().Format("2006-01-02 15:04:05"))

		taskSuccess := true

		for _, cfg := range *monCfgs {
			currentTime := time.Now()
			year := currentTime.Format("2006")
			month := currentTime.Format("01")
			day := currentTime.Format("02")

			dirPath := filepath.Join(cfg.SavePath, year, month)

			fmt.Printf("Searching for files to compress in directory: %s\n", dirPath)

			fileInfo, err := os.Stat(dirPath)
			if os.IsNotExist(err) {
				fmt.Printf("[ERROR] Path %s not found. Skipping: %v\n", dirPath, err)
				taskSuccess = false
				continue
			}
			if !fileInfo.IsDir() {
				fmt.Printf("[ERROR] Path %s is not a directory. Skipping.\n", dirPath)
				taskSuccess = false
				continue
			}

			files, err := ioutil.ReadDir(dirPath)
			if err != nil {
				fmt.Printf("[ERROR] error reading directory %s: %v\n", dirPath, err)
				taskSuccess = false
				continue
			}

			for _, file := range files {
				if file.IsDir() {
					continue
				}

				fileName := file.Name()
				filePath := filepath.Join(dirPath, fileName)

				if strings.HasSuffix(fileName, ".tar.gz") {
					fmt.Printf("-> file %s is already compressed, ignoring.\n", fileName)
					continue
				}

				parts := strings.Split(fileName, "_")
				if len(parts) < 2 {
					continue
				}

				numStr := strings.Split(parts[len(parts)-1], ".")[0]

				nDay, err := strconv.Atoi(day)
				if err != nil {
					fmt.Printf("[WARN] fail to convert current day '%s' to int: %v. skip file %s.\n", day, err, fileName)
					continue
				}

				nFileDay, err := strconv.Atoi(numStr)
				if err != nil {
					continue
				}

				diffDays := nDay - nFileDay

				if diffDays > 3 {
					//fmt.Printf("-> File %s (date: %s, %d days old) is older than 3 days. Initiating compression.\n", fileName, fileDate.Format("2006-01-02"), diffDays)

					//tar.gz
					compressedFilePath := filePath + ".tar.gz"
					err := compressFileTarGzip(filePath, compressedFilePath)
					if err != nil {
						fmt.Printf("[ERROR] Error compressing file %s: %v\n", fileName, err)
						taskSuccess = false
						continue
					}

					err = os.Remove(filePath)
					if err != nil {
						fmt.Printf("[ERROR] Error deleting original file %s: %v\n", fileName, err)
						taskSuccess = false
					} else {
						fmt.Printf("-> Original file %s delete. compressed file: %s\n", fileName, compressedFilePath)
					}
				} else {
					//fmt.Printf("-> File %s (date: %s, %d days old) is not a compression target.\n", fileName, fileDate.Format("2006-01-02"), diffDays)
				}
			}
		}

		if taskSuccess {
			fmt.Println("--- File compression task completed: succ ---")
		} else {
			fmt.Println("--- File compression task completed: error ---")
		}

		now := time.Now()
		nextRun := time.Date(now.Year(), now.Month(), now.Day()+1, 3, 0, 0, 0, now.Location())

		if now.After(nextRun) {
			nextRun = nextRun.Add(24 * time.Hour)
		}

		sleepDuration := nextRun.Sub(now)
		fmt.Printf("next compression task will run on %s. (time remaining: %s)\n", nextRun.Format("2006-01-02 15:04:05"), sleepDuration.String())

		time.Sleep(sleepDuration)
	}
}

// 압축
func compressFileTarGzip(srcPath, dstPath string) error {
	outputFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create compressed file: %w", err)
	}
	defer outputFile.Close()

	gzipWriter := gzip.NewWriter(outputFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	fileInfo, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("failed to get file info for %s: %w", srcPath, err)
	}

	header, err := tar.FileInfoHeader(fileInfo, "")
	if err != nil {
		return fmt.Errorf("failed to create tar header for %s: %w", srcPath, err)
	}

	header.Name = filepath.Base(srcPath)

	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header for %s: %w", srcPath, err)
	}

	inputFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", srcPath, err)
	}
	defer inputFile.Close()

	if _, err := io.Copy(tarWriter, inputFile); err != nil {
		return fmt.Errorf("failed to copy file content to tar archive for %s: %w", srcPath, err)
	}

	return nil
}
