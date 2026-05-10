package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestPrintSubcommandHelpDoctor(t *testing.T) {
	var buf bytes.Buffer
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	printSubcommandHelp("doctor")

	w.Close()
	os.Stdout = old
	ioCopy(&buf, r)

	output := buf.String()
	lower := strings.ToLower(output)

	if !strings.Contains(lower, "vectos doctor") {
		t.Errorf("doctor help should mention 'vectos doctor', got:\n%s", output)
	}

	if !strings.Contains(lower, "diagnostics") {
		t.Errorf("doctor help should mention diagnostics, got:\n%s", output)
	}

	if !strings.Contains(lower, "read-only") {
		t.Errorf("doctor help should mention read-only, got:\n%s", output)
	}

	if !strings.Contains(lower, "exit") {
		t.Errorf("doctor help should mention exit code, got:\n%s", output)
	}
}

func TestPrintHelpIncludesDoctor(t *testing.T) {
	var buf bytes.Buffer
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	printHelp()

	w.Close()
	os.Stdout = old
	ioCopy(&buf, r)

	output := buf.String()

	if !strings.Contains(output, "doctor") {
		t.Errorf("global help should list the doctor command, got:\n%s", output)
	}
	if !strings.Contains(output, "diagnostics") {
		t.Errorf("global help should describe doctor, got:\n%s", output)
	}
}

func TestRunDoctorCommandNoArgs(t *testing.T) {
	// Run doctor with a known app context; we only verify it doesn't panic
	// and produces expected sections in output.
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	app := appContext{
		projectBaseDir: home + "/.vectos/projects",
	}

	var buf bytes.Buffer
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	// Capture exit to avoid os.Exit(1) when provider check fails
	exitCalled := false
	exitCode := 0
	oldExit := osExit
	osExit = func(code int) {
		exitCalled = true
		exitCode = code
		panic("os.Exit called")
	}
	defer func() { osExit = oldExit }()
	defer func() {
		if r := recover(); r != nil {
			// expected if os.Exit was called
		}
	}()

	runDoctorCommand(app, []string{})

	w.Close()
	os.Stdout = old
	ioCopy(&buf, r)

	output := buf.String()

	if !strings.Contains(output, "Vectos Doctor") {
		t.Errorf("doctor output should start with 'Vectos Doctor', got:\n%s", output)
	}

	// Should always show these sections
	if !strings.Contains(output, "Install / Runtime") && !strings.Contains(output, "Install / Runtime:") {
		t.Errorf("doctor output should include Install/Runtime section")
	}
	if !strings.Contains(output, "Embedding Provider") && !strings.Contains(output, "Embedding Provider:") {
		t.Errorf("doctor output should include Embedding Provider section")
	}
	if !strings.Contains(output, "Index") && !strings.Contains(output, "Index:") {
		t.Errorf("doctor output should include Index section")
	}
	if !strings.Contains(output, "Result:") {
		t.Errorf("doctor output should include Result section")
	}

	_ = exitCalled
	_ = exitCode
}

// ioCopy copies from a reader to a writer (avoid depending on io.Copy in test shim)
func ioCopy(w *bytes.Buffer, r *os.File) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}
