package logsync

import (
	"fmt"
	"os"
	"strings"
	"strconv"
	"github.com/kengho/wdpath"
	"github.com/kengho/errors"
	"github.com/kengho/logs"
)

var MAX_LOG_BEGINNING_LENGTH = 256
var APPEND = true

func LogToBuf(logPath, logRotatePath, bufferPath string) () {
	logWatcherPath := getWatcherPath(logPath)
	logs.Logf("openOrCreateFile('%v')\n", logWatcherPath)
	logWatcherFile := openOrCreateFile(logWatcherPath, !APPEND)
	logs.Logf("readLogwatcher(openOrCreateFile('%v'))\n", logWatcherPath)
	savedOffset, savedLogBeginning := readLogwatcher(logWatcherFile)

	logFile, err := os.Open(logPath)
	errors.HandleErr(err)
	defer logFile.Close()
	logStat, err := logFile.Stat()
	errors.HandleErr(err)

	logs.Logf("getRealLogBeginning(os.Open('%v'))\n", logPath)
	realLogBeginning := getRealLogBeginning(logFile)
	logs.Logf("\trealLogBeginning = '%v'\n", realLogBeginning)

	var offset int64
	bytesToCmp := min(len(realLogBeginning), len(savedLogBeginning))
	if realLogBeginning[:bytesToCmp] == savedLogBeginning[:bytesToCmp] {
		logs.Logf("realLogBeginning[:%v] == '%v' == savedLogBeginning[:%v] == '%v'\n", bytesToCmp, realLogBeginning[:bytesToCmp], bytesToCmp, savedLogBeginning[:bytesToCmp])

		offset = savedOffset
	} else {
		logs.Logf("realLogBeginning[:%v] == '%v' != savedLogBeginning[:%v] == '%v'\n", bytesToCmp, realLogBeginning[:bytesToCmp], bytesToCmp, savedLogBeginning[:bytesToCmp])

		logRotateFile, err := os.Open(logRotatePath)
		defer logRotateFile.Close()
		if err == nil {
			_, _ = writeRestOfFileToBuffer(logRotateFile, savedOffset, bufferPath)
		}
		offset = 0
	}
	logs.Logf("offset = %v\n", offset)
	logs.Logf("logStat.Size() = %v\n", logStat.Size())

	if logStat.Size() - offset > 0 {
		bufferFile, bytesWritten := writeRestOfFileToBuffer(logFile, offset, bufferPath)

		logs.Logf("writeLogwatcher(logWatcherFile, (offset + bytesWritten) == %v, '%v')\n", offset + bytesWritten, realLogBeginning)
		writeLogwatcher(logWatcherFile, offset + bytesWritten, realLogBeginning)
		_ = bufferFile.Close()
	} else {
		logs.Logf("(logStat.Size() - offset) == %v <= 0, skipping\n", logStat.Size() - offset)
	}
}

func openOrCreateFile(path string, append bool) (*os.File) {
	var flags = os.O_RDWR | os.O_CREATE
	var flagsStr = "os.O_RDWR | os.O_CREATE"
	if append == true {
		flags = flags | os.O_APPEND
		flagsStr =flagsStr + " | os.O_APPEND"
	}
	logs.Logf("\tos.OpenFile('%v', %v, 0660)\n", path, flagsStr)
	f, err := os.OpenFile(path, flags, 0660)
	errors.HandleErr(err)
	return f
}

func readLogwatcher(logWatcherFile *os.File) (offset int64, logBeginning string) {
	logWatcherStat, err := logWatcherFile.Stat()
	errors.HandleErr(err)
	buf := make([]byte, logWatcherStat.Size())
	logs.Logf("\tlogWatcherStat.Size() = %v\n", logWatcherStat.Size())

	n, err := logWatcherFile.ReadAt(buf, 0)
	errors.HandleErr(err)
	logs.Logf("\tn = %v\n", n)
	logs.Logf("\tbuf = %v\n", buf)
	s := string(buf[:n])
	// empty file
	if s == "" {
		s = "0\n"
	}
	logs.Logf("\ts = '%v'\n", s)

	split := strings.Split(s, "\n")
	offset, err = strconv.ParseInt(split[0], 10, 64)
	errors.HandleErr(err)
	// join back all elements but first
	logBeginning = strings.Join(split[1:len(split)], "\n")
	logs.Logf("\toffset = %v\n", offset)
	logs.Logf("\tlogBeginning = '%v'\n", logBeginning)
	return offset, logBeginning
}

func writeLogwatcher(f *os.File, offset int64, realLogBeginning string) () {
	s := fmt.Sprintf("%d\n%s", offset, realLogBeginning)
	logs.Logf("\ts = '%v'\n", s)
	// completely rewrite file
	err := f.Truncate(0)
	errors.HandleErr(err)
	_, err = f.WriteAt([]byte(s), 0)
	errors.HandleErr(err)
}

func getRealLogBeginning(logFile *os.File) (string) {
	// @TODO figure out var naming
	logStat, err := logFile.Stat()
	errors.HandleErr(err)
	logs.Logf("\tmin64(logStat.Size() == %v, MAX_LOG_BEGINNING_LENGTH == %v) == %v\n", logStat.Size(), MAX_LOG_BEGINNING_LENGTH, min64(logStat.Size(), int64(MAX_LOG_BEGINNING_LENGTH)))
	buf := make([]byte, min64(logStat.Size(), int64(MAX_LOG_BEGINNING_LENGTH)))
	n, err := logFile.ReadAt(buf, 0)
	errors.HandleErr(err)
	logs.Logf("\tn = %v\n", n)
	return string(buf[:n])
}

func readRestOfFile(f *os.File, offset int64) (buf []byte) {
	// @TODO don't get Stat twice
	fStat, err := f.Stat()
	errors.HandleErr(err)
	logs.Logf("readRestOfFile(f.Name() == '%v', offset == %v)\n", fStat.Name(), offset)

	buf = make([]byte, fStat.Size() - offset)
	logs.Logf("buf is length %v\n", len(buf))

	logs.Logf("f.ReadAt(buf, offset == %v)\n", offset)
	bytesRed, err := f.ReadAt(buf, offset)
	errors.HandleErr(err)
	logs.Logf("\tbytesRed = %v\n", bytesRed)
	return buf
}

func saveToBuffer(bufferPath string, buf []byte) (*os.File, int64) {
	bufferFile := openOrCreateFile(bufferPath, APPEND)
	bytesWritten, err := bufferFile.Write(buf)
	errors.HandleErr(err)
	return bufferFile, int64(bytesWritten)
}

func writeRestOfFileToBuffer(f *os.File, offset int64, bufferPath string) (*os.File, int64) {
	buf := readRestOfFile(f, offset)
	logs.Logf("saveToBuffer(%v, buf)\n", bufferPath)
	return saveToBuffer(bufferPath, buf)
}

func getWatcherPath(path string) (string) {
	watcherPath := strings.Replace(path, "\\", "-", -1)
	watcherPath = strings.Replace(watcherPath, ":", ")", -1)
	watcherPath = fmt.Sprintf(".watcher_%s", watcherPath)
	watcherPath = wdpath.WdPath(watcherPath)
	return watcherPath
}

// http://stackoverflow.com/a/27516559
func min64(a, b int64) (int64) {
  if a < b {
    return a
  }
  return b
}

func min(a, b int) (int) {
  if a < b {
    return a
  }
  return b
}