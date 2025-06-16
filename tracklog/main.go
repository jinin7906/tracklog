package main

//추가할 기능
//디렉터리 단위 모니터링 기능 -> OK -> yyyy - mm - 파일명[hh].log 으로 경로 고정
//tracklog 로그파일 압축 -> 압축 로직 OK
//cpu, memory 사용제한 -> 홀딩
//실시간 처리 or 분기 처리 옵션 -> monitor에서 분기 처리

//추후에 DB 연동 옵션도 고려하자(퍼포먼스 잘나오는 성능좋은 DB찾아야함)
//UI도 구현 할까?
//프로세스 비정상 종료시 이어서 진행하는 방법이나 최초 실행시 처음부터 수집하는 방법도 생각 필요함 -> last line에다가 해당 파일을 체킹한 부분을 로그로 남기는 방식으로 하면 될거같긴함 다음꺼 이어서 쓸때 지우면되니까

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"

	"tracklog/config"
	"tracklog/manager"
	"tracklog/monitor"
	"tracklog/processor"
)

type MainThis struct {
	MonitorMgr   *monitor.MonitorMgr
	ProcessorMgr *processor.ProcessMgr
	DataMgr      *manager.Mgr
}

func main() {

	MainThis := new(MainThis)

	configPath := flag.String("config", "config/config.yaml", "config file path")
	flag.Parse()

	//read config
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load err: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("config load succ - %d\n", len(cfg.Monitors))

	MainThis.DataMgr = manager.NewMgr()

	var expandedMonitors []config.MonitorConfig

	for _, monCfg := range cfg.Monitors {
		if strings.Contains(monCfg.FilenamePattern, "*") || strings.Contains(monCfg.FilenamePattern, "?") {
			fileArr, err := MainThis.DataMgr.GetMatchingFiles(monCfg.Path, monCfg.FilenamePattern)
			if err != nil {
				fmt.Printf("fail find files match pattern (%s): %v\n", monCfg.FilenamePattern, err)
				continue
			}

			if len(fileArr) == 0 {
				fmt.Printf("skip monitoring. '%s'\n", monCfg.FilenamePattern)
				continue
			}
			for _, file := range fileArr {
				newMonCfg := monCfg
				newMonCfg.Name = fmt.Sprintf("%s_%s", monCfg.Name, strings.ReplaceAll(file, ".", "_"))
				newMonCfg.FilenamePattern = file
				expandedMonitors = append(expandedMonitors, newMonCfg)
				fmt.Printf("Added expanded monitor: Name='%s', File='%s'\n", newMonCfg.Name, newMonCfg.FilenamePattern)
			}
		} else {
			expandedMonitors = append(expandedMonitors, monCfg)
		}
	}
	cfg.Monitors = expandedMonitors
	fmt.Printf("Monitor configuration expansion complete. Total %d monitoring items.\n", len(cfg.Monitors))

	// monitor to processor
	var wg sync.WaitGroup
	logLineChan := make(chan manager.LogLine, cfg.Global.MaxGoroutines*2)
	logLineChanSch := make(chan manager.LogLine, cfg.Global.MaxGoroutines*2)

	MainThis.DataMgr.Start(&cfg.Monitors, &wg)

	MainThis.MonitorMgr = monitor.NewMonitorMgr(MainThis.DataMgr, &cfg.Global)
	MainThis.MonitorMgr.Start(&cfg.Monitors, logLineChan, logLineChanSch, &wg)

	MainThis.ProcessorMgr = processor.NewProcessMgr(MainThis.DataMgr, &cfg.Global, &cfg.Monitors)
	MainThis.ProcessorMgr.Start(logLineChan, &wg)

	fmt.Println("start Tracklog service")
	select {}

}
