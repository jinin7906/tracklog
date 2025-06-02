package manager

import (
	"fmt"
	"net"
	"path/filepath"
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
