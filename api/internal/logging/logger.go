package logging

import (
	"encoding/json"
	"log"
	"time"
)

type Logger struct {
	service string
}

func New(service string) *Logger { return &Logger{service: service} }

func (l *Logger) Info(msg string, fields map[string]interface{})  { l.write("INFO", msg, fields) }
func (l *Logger) Error(msg string, fields map[string]interface{}) { l.write("ERROR", msg, fields) }

func (l *Logger) write(level, msg string, fields map[string]interface{}) {
	if fields == nil {
		fields = map[string]interface{}{}
	}
	fields["ts"] = time.Now().UTC().Format(time.RFC3339Nano)
	fields["level"] = level
	fields["service"] = l.service
	fields["msg"] = msg
	b, _ := json.Marshal(fields)
	log.Println(string(b))
}
