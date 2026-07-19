package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewCreatesLogFile(t *testing.T) {
	// Redirect HOME to a temp dir so we don't touch the real ~/.lazytmux.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	lg, err := New("LAZYTMUX")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if lg == nil {
		t.Fatal("New returned nil logger")
	}

	// Writing should flush to the log file under tmp/.lazytmux/lazytmux.log.
	lg.Info("hello lazytmux")
	_ = lg.Sync()

	logPath := filepath.Join(tmp, ".lazytmux", "lazytmux.log")
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("log file is empty")
	}
}
