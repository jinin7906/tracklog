package manager

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
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

// // write tracklogS
// func (This *Mgr) AppendToFile(filePath, content string) error {
// 	dir := filepath.Dir(filePath)

// 	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
// 		return fmt.Errorf("dir create err %s: %w", dir, err)
// 	}

// 	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
// 	if err != nil {
// 		return fmt.Errorf("tracklog file open err: %s: %w", filePath, err)
// 	}
// 	defer f.Close()

// 	if _, err := f.WriteString(content); err != nil {
// 		return fmt.Errorf("tracklog file write err: %s: %w", filePath, err)
// 	}
// 	return nil
// }

// func (This *Mgr) AppendToFile(originalFilePath, content string) error {

// 	currentTime := time.Now()
// 	baseDir := filepath.Dir(originalFilePath)
// 	fileName := filepath.Base(originalFilePath)

// 	year := currentTime.Format("2006") // year
// 	month := currentTime.Format("01")  // month

// 	finalDir := filepath.Join(baseDir, year, month)

// 	dirReady, err := This.CreateDir(finalDir)
// 	if err != nil || !dirReady {
// 		return fmt.Errorf("failed to create directory for file %s: %w", finalDir, err)
// 	}

// 	timeSuffix := currentTime.Format("_15") // hour
// 	newFileName := fmt.Sprintf("%s%s", fileName, timeSuffix)

// 	finalFilePath := filepath.Join(finalDir, newFileName)

// 	f, err := os.OpenFile(finalFilePath, os.O_CREATE|os.O_WRONLY, 0644)
// 	if err != nil {
// 		return fmt.Errorf("tracklog file open err: %s: %w", finalFilePath, err)
// 	}
// 	defer f.Close()

// 	if _, err := f.WriteString(content); err != nil {
// 		return fmt.Errorf("tracklog file write err: %s: %w", finalFilePath, err)
// 	}

// 	return nil
// }

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

	timeSuffix := currentTime.Format("_15")

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

// write log
// func (This *Mgr) AppendLogContentToFile(baseDir string, content string) error {

// 	fmt.Printf("AA : [%s]\n", baseDir)

// 	currentTime := time.Now()
// 	filePath, err := This.GetDailyLogFilePath(baseDir, currentTime)
// 	if err != nil {
// 		return fmt.Errorf("failed to get log file path: %w", err)
// 	}

// 	fmt.Printf("BB : [%s]\n", filePath)

// 	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
// 	if err != nil {
// 		return fmt.Errorf("failed to open/create log file %s: %w", filePath, err)
// 	}
// 	defer file.Close()

// 	_, err = file.WriteString(content + "\n")
// 	if err != nil {
// 		return fmt.Errorf("failed to write to log file %s: %w", filePath, err)
// 	}

// 	return nil
// }
