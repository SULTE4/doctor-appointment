package logger

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

type Logger struct {
	stdout *json.Encoder
	stderr *json.Encoder
	mu     sync.Mutex
}

func New() *Logger {
	return &Logger{
		stdout: json.NewEncoder(os.Stdout),
		stderr: json.NewEncoder(os.Stderr),
	}
}

func (l *Logger) Event(subject string, event any) {
	l.writeStdout(map[string]any{
		"time":    time.Now().UTC().Format(time.RFC3339),
		"subject": subject,
		"event":   event,
	})
}

func (l *Logger) Job(level, jobID string, attempt int, status string, err error) {
	entry := map[string]any{
		"time":    time.Now().UTC().Format(time.RFC3339),
		"level":   level,
		"job_id":  jobID,
		"attempt": attempt,
		"status":  status,
	}
	if err != nil {
		entry["error"] = err.Error()
	}

	if status == "dead_letter" {
		l.writeStderr(entry)
		return
	}

	l.writeStdout(entry)
}

func (l *Logger) writeStdout(entry map[string]any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.stdout.Encode(entry); err != nil {
		log.Printf("[ERROR] failed to write stdout json log: %v", err)
	}
}

func (l *Logger) writeStderr(entry map[string]any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.stderr.Encode(entry); err != nil {
		log.Printf("[ERROR] failed to write stderr json log: %v", err)
	}
}
