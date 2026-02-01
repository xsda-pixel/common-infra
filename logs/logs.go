package logs

import (
	"io"
	"os"
	"sync"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
)

var (
	LogFilePath      = "./logs/"            // 日志保存路径
	Logger           *logrus.Logger         // 初始化日志对象
	DbWriter         *rotatelogs.RotateLogs // 初始化日志对象
	GinDefaultWriter *rotatelogs.RotateLogs // 初始化日志对象
	GinErrWriter     *rotatelogs.RotateLogs // 初始化日志对象
	once             sync.Once
)

func Init(isGin, isDb, debug bool) error {
	var initErr error
	once.Do(func() {
		initErr = doInit(isGin, isDb, debug)
	})
	return initErr
}

const (
	defaultLogMaxAge   = 7 * 24 * time.Hour
	defaultLogRotation = 24 * time.Hour
)

// newRotateLogs 创建按天切割的日志 writer，默认保留 7 天
func newRotateLogs(path string) (*rotatelogs.RotateLogs, error) {
	return rotatelogs.New(
		path,
		rotatelogs.WithMaxAge(defaultLogMaxAge),
		rotatelogs.WithRotationTime(defaultLogRotation),
	)
}

func doInit(isGin, isDb, debug bool) error {
	var (
		err        error
		logWriter  *rotatelogs.RotateLogs
		errWriter  *rotatelogs.RotateLogs
		infoWriter *rotatelogs.RotateLogs
		dirs       = []string{
			LogFilePath,
			LogFilePath + "gins",
			LogFilePath + "dbs",
			LogFilePath + "errors",
			LogFilePath + "infos",
			LogFilePath + "default",
		}
	)

	for _, dir := range dirs {
		if err = os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
	}

	logger := logrus.New()
	if debug {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	if isGin {
		GinDefaultWriter, err = newRotateLogs(LogFilePath + "gins/default-%Y%m%d.log")
		if err != nil {
			return err
		}
		GinErrWriter, err = newRotateLogs(LogFilePath + "gins/err-%Y%m%d.log")
		if err != nil {
			return err
		}
	}

	logWriter, err = newRotateLogs(LogFilePath + "default/log-%Y%m%d.log")
	if err != nil {
		return err
	}

	if isDb {
		DbWriter, err = newRotateLogs(LogFilePath + "dbs/log-%Y%m%d.log")
		if err != nil {
			return err
		}
	}

	errWriter, err = newRotateLogs(LogFilePath + "errors/err-%Y%m%d.log")
	if err != nil {
		return err
	}

	infoWriter, err = newRotateLogs(LogFilePath + "infos/info-%Y%m%d.log")
	if err != nil {
		return err
	}

	logger.SetReportCaller(debug) // 代码文件的绝对路径、方法名、行号

	writeMap := lfshook.WriterMap{
		logrus.DebugLevel: infoWriter,
		logrus.InfoLevel:  infoWriter,
		logrus.WarnLevel:  logWriter,
		logrus.ErrorLevel: errWriter,
		logrus.FatalLevel: errWriter,
		logrus.PanicLevel: errWriter,
	}

	logger.AddHook(lfshook.NewHook(writeMap, &logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "msg",
			logrus.FieldKeyTime:  "time",
			logrus.FieldKeyFile:  "caller",
		},
	}))

	logger.SetOutput(io.Discard) // 控制台不打印
	Logger = logger
	return nil
}
