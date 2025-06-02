package main

//추가할 기능
//디렉터리 단위 모니터링 기능
//tracklog 로그파일 압축
//cpu, memory 사용제한
//실시간 처리 or 분기 처리 옵션

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

	// for _, monCfg := range cfg.Monitors {
	// 	fileArr, err := MainThis.DataMgr.GetMatchingFiles(monCfg.Path, monCfg.FilenamePattern)
	// 	if err != nil {
	// 		fmt.Printf("[MonitorMgr] Start - not find file")
	// 	}
	// 	tmpMonCfg := monCfg.Name
	// 	for _, file := range fileArr {
	// 		//여기서 배열에 추가하면 될거같은데
	// 	}
	// }

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

	MainThis.MonitorMgr = monitor.NewMonitorMgr(MainThis.DataMgr, &cfg.Global)
	MainThis.MonitorMgr.Start(&cfg.Monitors, logLineChan, &wg)

	MainThis.ProcessorMgr = processor.NewProcessMgr(MainThis.DataMgr, &cfg.Global, &cfg.Monitors)
	MainThis.ProcessorMgr.Start(logLineChan, &wg)

	fmt.Println("start Tracklog service")
	select {}

}
