package logs

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/lestrrat-go/file-rotatelogs"
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

	// 实例化
	logger := logrus.New()
	// 设置日志级别
	if debug {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	if isGin {
		// 设置 rotatelogs
		GinDefaultWriter, err = rotatelogs.New(
			// 分割后的文件名称
			LogFilePath+"gins/default-%Y%m%d.log",

			// 设置最大保存时间(7天)
			rotatelogs.WithMaxAge(7*24*time.Hour),

			// 设置日志切割时间间隔(1天)
			rotatelogs.WithRotationTime(24*time.Hour),
		)
		if err != nil {
			return err
		}
		GinErrWriter, err = rotatelogs.New(
			// 分割后的文件名称
			LogFilePath+"gins/err-%Y%m%d.log",

			// 设置最大保存时间(7天)
			rotatelogs.WithMaxAge(7*24*time.Hour),

			// 设置日志切割时间间隔(1天)
			rotatelogs.WithRotationTime(24*time.Hour),
		)
		if err != nil {
			return err
		}
	}

	logWriter, err = rotatelogs.New(
		// 分割后的文件名称
		LogFilePath+"default/log-%Y%m%d.log",

		// 设置最大保存时间(7天)
		rotatelogs.WithMaxAge(7*24*time.Hour),

		// 设置日志切割时间间隔(1天)
		rotatelogs.WithRotationTime(24*time.Hour),
	)
	if err != nil {
		return err
	}

	if isDb {
		// 设置 rotatelogs
		DbWriter, err = rotatelogs.New(
			// 分割后的文件名称
			LogFilePath+"dbs/log-%Y%m%d.log",

			// 设置最大保存时间(7天)
			rotatelogs.WithMaxAge(7*24*time.Hour),

			// 设置日志切割时间间隔(1天)
			rotatelogs.WithRotationTime(24*time.Hour),
		)
		if err != nil {
			return err
		}
	}

	// 设置 rotatelogs
	errWriter, err = rotatelogs.New(
		// 分割后的文件名称
		LogFilePath+"errors/err-%Y%m%d.log",

		// 设置最大保存时间(7天)
		rotatelogs.WithMaxAge(7*24*time.Hour),

		// 设置日志切割时间间隔(1天)
		rotatelogs.WithRotationTime(24*time.Hour),
	)
	if err != nil {
		return err
	}

	// 设置 rotatelogs
	infoWriter, err = rotatelogs.New(
		// 分割后的文件名称
		LogFilePath+"infos/info-%Y%m%d.log",

		// 生成软链，指向最新日志文件
		// rotatelogs.WithLinkName(fileName),

		// 设置最大保存时间(7天)
		rotatelogs.WithMaxAge(7*24*time.Hour),

		// 设置日志切割时间间隔(1天)
		rotatelogs.WithRotationTime(24*time.Hour),
	)
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
