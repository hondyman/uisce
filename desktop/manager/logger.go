package manager

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

// WorkstationLogWriter provides a rotating, size-capped log file with strict
// secret scrubbing to prevent credential leakage.
type WorkstationLogWriter struct {
	mu       sync.Mutex
	filePath string
	maxBytes int64
	file     *os.File
	curSize  int64
}

// Regex patterns to scrub sensitive authentication parameters, JWTs, and keys
var (
	bearerRegex = regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9-_=]+\.[A-Za-z0-9-_=]+\.?[A-Za-z0-9-_.+/=]*`)
	jwtRegex    = regexp.MustCompile(`eyJ[A-Za-z0-9-_=]+\.eyJ[A-Za-z0-9-_=]+\.?[A-Za-z0-9-_.+/=]*`)
	tokenRegex  = regexp.MustCompile(`(?i)(init_token|token|secret|password|jwt)=([A-Za-z0-9-_=]+)`)
)

// ScrubSecrets replaces any detected credential strings with a redaction marker.
func ScrubSecrets(msg string) string {
	scrubbed := bearerRegex.ReplaceAllString(msg, "Bearer [REDACTED_JWT]")
	scrubbed = jwtRegex.ReplaceAllString(scrubbed, "[REDACTED_JWT]")
	scrubbed = tokenRegex.ReplaceAllString(scrubbed, "$1=[REDACTED]")
	return scrubbed
}

// ScrubbingWriter wraps an underlying io.Writer and scrubs all written bytes before emitting.
type ScrubbingWriter struct {
	target io.Writer
}

func (s *ScrubbingWriter) Write(p []byte) (n int, err error) {
	scrubbed := ScrubSecrets(string(p))
	_, err = s.target.Write([]byte(scrubbed))
	return len(p), err
}

// InitWorkstationLogger sets up persistent, rotating local logging under
// ~/Library/Logs/Uisce/workstation.log with a 10MB rotation ceiling.
// Output is teed to stderr, with both streams passing through ScrubSecrets.
func InitWorkstationLogger() io.Closer {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	logDir := filepath.Join(homeDir, "Library", "Logs", "Uisce")
	_ = os.MkdirAll(logDir, 0755)

	logPath := filepath.Join(logDir, "workstation.log")
	writer, err := NewRotatingLogWriter(logPath, 10*1024*1024) // 10 MB cap
	if err != nil {
		log.Printf("[Logger] Warning: Could not initialize rotating log file: %v", err)
		return nil
	}

	// Route both stderr and the rotating file through secret scrubbing
	scrubbedStderr := &ScrubbingWriter{target: os.Stderr}
	multi := io.MultiWriter(scrubbedStderr, writer)
	log.SetOutput(multi)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lmsgprefix)
	log.Printf("[WorkstationLogger] Local logging initialized at %s (max 10MB rotating, 1 backup window)", logPath)

	return writer
}

// NewRotatingLogWriter creates a new rotating log writer.
func NewRotatingLogWriter(filePath string, maxBytes int64) (*WorkstationLogWriter, error) {
	w := &WorkstationLogWriter{
		filePath: filePath,
		maxBytes: maxBytes,
	}

	if err := w.openFile(); err != nil {
		return nil, err
	}

	return w, nil
}

func (w *WorkstationLogWriter) openFile() error {
	f, err := os.OpenFile(w.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}

	w.file = f
	w.curSize = info.Size()
	return nil
}

// Write scrubs secrets and writes to the log file, rotating when maxBytes is reached.
func (w *WorkstationLogWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	scrubbed := ScrubSecrets(string(p))
	data := []byte(scrubbed)

	if w.curSize+int64(len(data)) > w.maxBytes {
		w.rotate()
	}

	if w.file == nil {
		if err := w.openFile(); err != nil {
			return 0, err
		}
	}

	n, err = w.file.Write(data)
	w.curSize += int64(n)
	return len(p), err
}

func (w *WorkstationLogWriter) rotate() {
	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}

	backupPath := w.filePath + ".1"
	_ = os.Remove(backupPath)
	_ = os.Rename(w.filePath, backupPath)
	w.curSize = 0
	_ = w.openFile()
}

// Close closes the underlying log file.
func (w *WorkstationLogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		err := w.file.Close()
		w.file = nil
		return err
	}
	return nil
}

// WriteCrashDump writes an exception diagnostic dump with secret scrubbing.
func WriteCrashDump(reason string, stackTrace string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	crashDir := filepath.Join(homeDir, "Library", "Logs", "Uisce", "crashes")
	_ = os.MkdirAll(crashDir, 0755)

	dumpFile := filepath.Join(crashDir, fmt.Sprintf("crash_%d.log", time.Now().Unix()))
	content := fmt.Sprintf("TIME: %s\nREASON: %s\nSTACK:\n%s\n",
		time.Now().Format(time.RFC3339),
		ScrubSecrets(reason),
		ScrubSecrets(stackTrace),
	)

	// Cap dump size to 64KB max
	if len(content) > 64*1024 {
		content = content[:64*1024] + "\n[TRUNCATED_AT_64KB]\n"
	}

	_ = os.WriteFile(dumpFile, []byte(content), 0644)
	return dumpFile
}
