package output

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestPrintMainHelp(t *testing.T) {
	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintMainHelp()

	w.Close()
	os.Stdout = old

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains expected text
	expectedStrings := []string{
		"MCPBridge",
		"Usage:",
		"mcpbridgego",
		"start",
		"stop",
		"help",
	}

	for _, str := range expectedStrings {
		if !bytes.Contains([]byte(output), []byte(str)) {
			t.Errorf("expected help output to contain '%s'", str)
		}
	}
}
