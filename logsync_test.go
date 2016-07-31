package logsync_test

import (
	"testing"
	"github.com/kengho/logsync"
	"os"
	"io/ioutil"
	"path/filepath"
	"bytes"
)

func TestLogsync(t *testing.T) {
	tmpDir, _ := ioutil.TempDir("", "logsync")
	logPath := filepath.Join(tmpDir, "log")
	logRotatePath := filepath.Join(tmpDir, "loglotate")
	bufferPath := filepath.Join(tmpDir, "buffer")

	var gs, es string

	chunk01 := "llllog_line01\n"
	writeChunk(logPath, chunk01)
	logsync.LogToBuf(logPath, logRotatePath, bufferPath)
	gs = readFile(bufferPath)
	es = chunk01
	if gs != es {
		t.Errorf("Expected gs to be '%v', got '%v'", es, gs)
	}

	chunk02 := "log_line02\n"
	writeChunk(logPath, chunk02)
	logsync.LogToBuf(logPath, logRotatePath, bufferPath)
	gs = readFile(bufferPath)
	es = concat([]string{chunk01, chunk02})
	if gs != es {
		t.Errorf("Expected gs to be '%v', got '%v'", es, gs)
	}

	chunk03 :=
		"log_linelinelinelinelinelinelinelinelineline03\n" +
		"log_linelinelinelinelinelinelinelinelineline04\n" +
		"log_linelinelinelinelinelinelinelinelineline05\n" +
		"log_linelinelinelinelinelinelinelinelineline06\n"
	writeChunk(logPath, chunk03)
	logsync.LogToBuf(logPath, logRotatePath, bufferPath)
	gs = readFile(bufferPath)
	es = concat([]string{es, chunk03})
	if gs != es {
		t.Errorf("Expected gs to be '%v', got '%v'", es, gs)
	}

	chunk04 := "log_overflow\n"
	rotatedChunk := concat([]string{es, chunk04})
	writeChunk(logRotatePath, rotatedChunk)
	os.Remove(logPath)
	chunk05 := "new_log_line01\n"
	writeChunk(logPath, chunk05)
	logsync.LogToBuf(logPath, logRotatePath, bufferPath)
	es = concat([]string{es, chunk04, chunk05})
	gs = readFile(bufferPath)
	if gs != es {
		t.Errorf("Expected gs to be '%v', got '%v'", es, gs)
	}

	os.RemoveAll(tmpDir)
}

func writeChunk(path string, line string) () {
	f, _ := os.OpenFile(path, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0660)
	f.Write([]byte(line))
	f.Close()
}

func readFile(path string) (string) {
	f, _ := os.Open(path)
	fStat, _ := f.Stat()
	buf := make([]byte, fStat.Size())
	n, _ := f.Read(buf)
	f.Close()
	return string(buf[:n])
}

func concat(arr []string) (string) {
	var buf bytes.Buffer
	for i := 0; i < len(arr); i++ {
		buf.WriteString(arr[i])
	}
	return buf.String()
}
