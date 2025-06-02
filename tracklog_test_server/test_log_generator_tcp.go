package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"time"
)

func handleConnection(conn net.Conn) {
	fmt.Printf("[TCP Server] 새로운 클라이언트 연결됨: %s\n", conn.RemoteAddr().String())
	defer func() {
		fmt.Printf("[TCP Server] 클라이언트 연결 종료됨: %s\n", conn.RemoteAddr().String())
		conn.Close()
	}()

	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("[TCP Server] 연결 읽기 오류: %v\n", err)
			break
		}

		fmt.Printf("[TCP Server] 수신: %s", message)
	}
}

func main() {
	logFilePath := flag.String("path", "/tmp/tracklog_test.log", "생성할 로그 파일 경로")
	intervalSec := flag.Int("interval", 1, "로그를 추가할 간격 (초)")
	logType := flag.String("type", "normal", "로그 타입: 'normal' 또는 'mysql_error' 또는 'mysql_slow'")
	tcpListenAddr := flag.String("tcp-listen", "127.0.0.1:9000", "TCP 서버가 리스닝할 주소")
	flag.Parse()

	go func() {
		fmt.Printf("[TCP Server] %s에서 수신 대기 중...\n", *tcpListenAddr)
		listener, err := net.Listen("tcp", *tcpListenAddr)
		if err != nil {
			fmt.Printf("[TCP Server] 리스너 시작 오류: %v\n", err)
			return
		}
		defer listener.Close()

		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Printf("[TCP Server] 연결 수락 오류: %v\n", err)
				continue
			}
			go handleConnection(conn)
		}
	}()

	file, err := os.OpenFile(*logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("로그 파일 열기/생성 오류: %v\n", err)
		return
	}
	defer file.Close()

	fmt.Printf("'%s' 파일에 %d초 간격으로 로그를 기록합니다 (타입: %s).\n", *logFilePath, *intervalSec, *logType)
	fmt.Println("종료하려면 Ctrl+C를 누르세요.")

	for {
		logLine := ""
		timestamp := time.Now().Format("2006-01-02 15:04:05")

		switch *logType {
		case "mysql_error":
			errorTypes := []string{"ERROR", "Warning", "Note"}
			errorMsg := []string{
				"Could not open file",
				"Access denied for user",
				"Aborted connection",
				"Out of memory",
				"Disk full",
			}
			randomErrorType := errorTypes[rand.Intn(len(errorTypes))]
			randomErrorMsg := errorMsg[rand.Intn(len(errorMsg))]
			logLine = fmt.Sprintf("%s [%s] [Server] [ERROR] %s\n", timestamp, randomErrorType, randomErrorMsg)
			if randomErrorType == "ERROR" && rand.Intn(10) < 3 {
				logLine = fmt.Sprintf("%s [%s] [Server] [ERROR] Failed password for 'user_%d'@'192.168.1.%d'\n",
					timestamp, randomErrorType, rand.Intn(100), rand.Intn(255))
			}
		case "mysql_slow":
			queryTypes := []string{"SELECT", "UPDATE", "INSERT", "DELETE"}
			randomQueryType := queryTypes[rand.Intn(len(queryTypes))]
			queryTime := fmt.Sprintf("%.6f", rand.Float64()*5+1)
			userId := rand.Intn(1000)
			hostIp := fmt.Sprintf("192.168.1.%d", rand.Intn(255))
			dbName := "mydb"
			tableName := fmt.Sprintf("my_table_%d", rand.Intn(5))

			logLine = fmt.Sprintf(`# Time: %s
# User@Host: user%d[%s] @ %s [%s]  Id: %d
# Query_time: %s  Lock_time: 0.000000 Rows_sent: %d  Rows_examined: %d
SET timestamp=%d;
USE %s;
%s * FROM %s WHERE id = %d;\n`,
				time.Now().Format("240102 15:04:05"),
				userId, "user_app", hostIp, hostIp, rand.Intn(10000),
				queryTime, rand.Intn(100), rand.Intn(1000),
				time.Now().Unix(),
				dbName,
				randomQueryType, tableName, rand.Intn(100000))
		case "normal":
			fallthrough
		default:
			logPrefix := fmt.Sprintf("%s INFO: ", timestamp)
			messages := []string{
				"This is a normal log message.",
				"User 'testuser' logged in successfully from 192.168.0.100.",
				"Processing data for task ID 12345.",
				"Failed to connect to database: Connection refused.",
				"HTTP/1.1\" 200 1234 /index.html",
				"HTTP/1.1\" 404 0 /nonexistent_page.html",
				"Failed password for user guest from 10.0.0.5",
				"CRITICAL ERROR: Disk space low on /dev/sda1.",
			}
			logLine = logPrefix + messages[rand.Intn(len(messages))] + "\n"
		}

		_, err = file.WriteString(logLine)
		if err != nil {
			fmt.Printf("로그 쓰기 오류: %v\n", err)
			return
		}

		time.Sleep(time.Duration(*intervalSec) * time.Second)
	}
}
