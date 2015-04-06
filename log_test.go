package logging

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func isFileExists(name string) bool {
	f, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}

	if f.IsDir() {
		return false
	}

	return true
}

func parseDate(value string, format string) (time.Time, error) {
	tt, err := time.ParseInLocation(format, value, time.Local)
	if err != nil {
		fmt.Println("[Error]" + err.Error())
		return tt, err
	}

	return tt, nil
}

func checkLogData(fileName string, containData string, num int64) error {
	input, err := os.OpenFile(fileName, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer input.Close()

	var lineNum int64
	br := bufio.NewReader(input)
	for {
		line, err := br.ReadString('\n')
		if err == io.EOF {
			break
		}

		realLine := strings.TrimRight(line, "\n")
		if strings.Contains(realLine, containData) {
			lineNum += 1
		}
	}

	// check whether num is equal to lineNum
	if lineNum != num {
		return fmt.Errorf("checkLogData fail - %d vs %d", lineNum, num)
	}

	return nil
}

func TestDayRotateCase(t *testing.T) {
	_b := NewSimpleBackend()

	logName := "example_day_test.log"
	if isFileExists(logName) {
		err := os.Remove(logName)
		if err != nil {
			t.Errorf("Remove old log file fail - %s, %s\n", err.Error(), logName)
		}
	}

	_b.SetRotateByDay()
	err := _b.SetOutputByName(logName)
	if err != nil {
		t.Errorf("SetOutputByName fail - %s, %s\n", err.Error(), logName)
	}

	if _b.logSuffix == "" {
		t.Errorf("bad log suffix fail - %s\n", _b.logSuffix)
	}

	day, err := parseDate(_b.logSuffix, FORMAT_TIME_DAY)
	if err != nil {
		t.Errorf("parseDate fail - %s, %s\n", err.Error(), _b.logSuffix)
	}

	_b.Info("Test data")
	_b.Infof("Test data - %s", day.String())

	// mock log suffix to check rotate
	lastDay := day.AddDate(0, 0, -1)
	_b.logSuffix = genDayTime(lastDay)
	oldLogSuffix := _b.logSuffix

	_b.Info("Test new data")
	_b.Infof("Test new data - %s", day.String())

	err = _b.fd.Close()
	if err != nil {
		t.Errorf("close log fd fail - %s, %s\n", err.Error(), _b.fileName)
	}

	// check both old and new log file datas
	oldLogName := logName + "." + oldLogSuffix
	err = checkLogData(oldLogName, "Test data", 2)
	if err != nil {
		t.Errorf("old log file checkLogData fail - %s, %s\n", err.Error(), oldLogName)
	}

	err = checkLogData(logName, "Test new data", 2)
	if err != nil {
		t.Errorf("new log file checkLogData fail - %s, %s\n", err.Error(), logName)
	}

	// remove test log files
	err = os.Remove(oldLogName)
	if err != nil {
		t.Errorf("Remove final old log file fail - %s, %s\n", err.Error(), oldLogName)
	}

	err = os.Remove(logName)
	if err != nil {
		t.Errorf("Remove final new log file fail - %s, %s\n", err.Error(), logName)
	}
}

func TestHourRotateCase(t *testing.T) {
	_b := NewSimpleBackend()

	logName := "example_hour_test.log"
	if isFileExists(logName) {
		err := os.Remove(logName)
		if err != nil {
			t.Errorf("Remove old log file fail - %s, %s\n", err.Error(), logName)
		}
	}

	_b.SetRotateByHour()
	err := _b.SetOutputByName(logName)
	if err != nil {
		t.Errorf("SetOutputByName fail - %s, %s\n", err.Error(), logName)
	}

	if _b.logSuffix == "" {
		t.Errorf("bad log suffix fail - %s\n", _b.logSuffix)
	}

	hour, err := parseDate(_b.logSuffix, FORMAT_TIME_HOUR)
	if err != nil {
		t.Errorf("parseDate fail - %s, %s\n", err.Error(), _b.logSuffix)
	}

	_b.Info("Test data")
	_b.Infof("Test data - %s", hour.String())

	// mock log suffix to check rotate
	lastHour := hour.Add(time.Duration(-1 * time.Hour))
	_b.logSuffix = genHourTime(lastHour)
	oldLogSuffix := _b.logSuffix

	_b.Info("Test new data")
	_b.Infof("Test new data - %s", hour.String())

	err = _b.fd.Close()
	if err != nil {
		t.Errorf("close log fd fail - %s, %s\n", err.Error(), _b.fileName)
	}

	// check both old and new log file datas
	oldLogName := logName + "." + oldLogSuffix
	err = checkLogData(oldLogName, "Test data", 2)
	if err != nil {
		t.Errorf("old log file checkLogData fail - %s, %s\n", err.Error(), oldLogName)
	}

	err = checkLogData(logName, "Test new data", 2)
	if err != nil {
		t.Errorf("new log file checkLogData fail - %s, %s\n", err.Error(), logName)
	}

	// remove test log files
	err = os.Remove(oldLogName)
	if err != nil {
		t.Errorf("Remove final old log file fail - %s, %s\n", err.Error(), oldLogName)
	}

	err = os.Remove(logName)
	if err != nil {
		t.Errorf("Remove final new log file fail - %s, %s\n", err.Error(), logName)
	}
}

func TestSizeRotateCase(t *testing.T) {
	_b := NewSimpleBackend()

	logName := "example_size_test.log"
	if isFileExists(logName) {
		err := os.Remove(logName)
		if err != nil {
			t.Errorf("Remove old log file fail - %s, %s\n", err.Error(), logName)
		}
	}

	_b.SetRotateBySize(int64(128))
	err := _b.SetOutputByName(logName)
	if err != nil {
		t.Errorf("SetOutputByName fail - %s, %s\n", err.Error(), logName)
	}

	if _b.logSuffix == "" || _b.logSuffix != "0" {
		t.Errorf("bad log suffix fail - %s\n", _b.logSuffix)
	}

	_b.Info("Test data")
	_b.Infof("Test data - %128s", "128 bytes mocked string")

	// mock log suffix to check rotate
	_b.logSuffix = genNextSeq("999")
	oldLogSuffix := _b.logSuffix

	_b.Info("Test new data")
	_b.Infof("Test new data - %128s", "128 bytes mocked new string")

	err = _b.fd.Close()
	if err != nil {
		t.Errorf("close log fd fail - %s, %s\n", err.Error(), _b.fileName)
	}

	// check both old and new log file datas
	oldLogName := logName + "." + oldLogSuffix
	err = checkLogData(oldLogName, "Test data", 2)
	if err != nil {
		t.Errorf("old log file checkLogData fail - %s, %s\n", err.Error(), oldLogName)
	}

	err = checkLogData(logName, "Test new data", 2)
	if err != nil {
		t.Errorf("new log file checkLogData fail - %s, %s\n", err.Error(), logName)
	}

	// remove test log files
	err = os.Remove(oldLogName)
	if err != nil {
		t.Errorf("Remove final old log file fail - %s, %s\n", err.Error(), oldLogName)
	}

	err = os.Remove(logName)
	if err != nil {
		t.Errorf("Remove final new log file fail - %s, %s\n", err.Error(), logName)
	}
}
