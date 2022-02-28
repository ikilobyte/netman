package util

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

var Logger = NewLogger()

//NewLogger 日志
func NewLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetReportCaller(true)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05.0000",
	})
	return logger
}

//MaxListenerBacklog 获取Accept队列的最大值
func MaxListenerBacklog() int {

	fd, err := os.Open("/proc/sys/net/core/somaxconn")
	if err != nil {
		return unix.SOMAXCONN
	}
	defer fd.Close()

	rd := bufio.NewReader(fd)
	line, err := rd.ReadString('\n')
	if err != nil {
		return unix.SOMAXCONN
	}

	f := strings.Fields(line)
	if len(f) < 1 {
		return unix.SOMAXCONN
	}

	n, err := strconv.Atoi(f[0])
	if err != nil || n == 0 {
		return unix.SOMAXCONN
	}
	if n > 1<<16-1 {
		n = 1<<16 - 1
	}
	return n
}
