package config

import (
	"log"

	"gopkg.in/natefinch/lumberjack.v2"
)

func InitLogger() {
	log.SetOutput(&lumberjack.Logger{
		Filename:   "./app.log",
		MaxSize:    100,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	})

	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
