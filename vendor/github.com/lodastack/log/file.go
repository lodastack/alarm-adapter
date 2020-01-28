package log

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"sync"
	"time"
)

const (
	bufferSize = 256 * 1024
)

type syncBuffer struct {
	*bufio.Writer
	file     *os.File
	count    uint64
	cur      int
	filePath string
	parent   *FileBackend
}

func (self *syncBuffer) Sync() error {
	return self.file.Sync()
}

func (self *syncBuffer) close() {
	self.Flush()
	self.Sync()
	self.file.Close()
}

func (self *syncBuffer) write(b []byte) {
	if self.parent.maxSize > 0 && self.parent.rotateNum > 0 && self.count+uint64(len(b)) >= self.parent.maxSize {
		os.Rename(self.filePath, self.filePath+fmt.Sprintf(".%03d", self.cur))
		self.cur++
		if self.cur >= self.parent.rotateNum {
			self.cur = 0
		}
		self.count = 0
	}
	self.count += uint64(len(b))
	self.Writer.Write(b)
}

type FileBackend struct {
	mu            sync.Mutex
	dir           string //directory for log files
	files         [numSeverity]syncBuffer
	flushInterval time.Duration
	rotateNum     int
	maxSize       uint64
	fall          bool
}

func (self *FileBackend) Flush() {
	self.mu.Lock()
	defer self.mu.Unlock()
	for i := 0; i < numSeverity; i++ {
		self.files[i].Flush()
		self.files[i].Sync()
	}
}

func (self *FileBackend) close() {
	self.Flush()
}

func (self *FileBackend) flushDaemon() {
	for {
		time.Sleep(self.flushInterval)
		self.Flush()
	}
}

func (self *FileBackend) monitorFiles() {
	for range time.NewTicker(time.Second * 5).C {
		for i := 0; i < numSeverity; i++ {
			fileName := path.Join(self.dir, severityName[i]+".log")
			if _, err := os.Stat(fileName); err != nil && os.IsNotExist(err) {
				if f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
					self.mu.Lock()
					self.files[i].close()
					self.files[i].Writer = bufio.NewWriterSize(f, bufferSize)
					self.files[i].file = f
					self.mu.Unlock()
				}
			}
		}
	}
}

func (self *FileBackend) Log(s Severity, msg []byte) {
	self.mu.Lock()
	switch s {
	case FATAL:
		self.files[FATAL].write(msg)
	case ERROR:
		self.files[ERROR].write(msg)
	case WARNING:
		self.files[WARNING].write(msg)
	case INFO:
		self.files[INFO].write(msg)
	case DEBUG:
		self.files[DEBUG].write(msg)
	}
	if self.fall && s < INFO {
		self.files[INFO].write(msg)
	}
	self.mu.Unlock()
	if s == FATAL {
		self.Flush()
	}
}

func (self *FileBackend) Rotate(rotateNum1 int, maxSize1 uint64) {
	self.rotateNum = rotateNum1
	self.maxSize = maxSize1
}

func (self *FileBackend) Fall() {
	self.fall = true
}

func (self *FileBackend) SetFlushDuration(t time.Duration) {
	if t >= time.Second {
		self.flushInterval = t
	} else {
		self.flushInterval = time.Second
	}
}

func NewFileBackend(dir string) (*FileBackend, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	var fb FileBackend
	fb.dir = dir
	for i := 0; i < numSeverity; i++ {
		fileName := path.Join(dir, severityName[i]+".log")
		f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		fb.files[i] = syncBuffer{Writer: bufio.NewWriterSize(f, bufferSize), file: f, filePath: fileName, parent: &fb}
	}
	// default
	fb.flushInterval = time.Second * 3
	fb.rotateNum = 20
	fb.maxSize = 1024 * 1024 * 1024
	fb.fall = false

	go fb.flushDaemon()
	go fb.monitorFiles()
	return &fb, nil
}

func Rotate(rotateNum1 int, maxSize1 uint64) {
	if fileback != nil {
		fileback.Rotate(rotateNum1, maxSize1)
	}
}

func Fall() {
	if fileback != nil {
		fileback.Fall()
	}
}

func SetFlushDuration(t time.Duration) {
	if fileback != nil {
		fileback.SetFlushDuration(t)
	}
}
