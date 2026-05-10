package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Options controls setup behavior.
type Options struct {
	Uninstall    bool
	SkipGuidance bool
	AssumeYes    bool
}

type Adapter interface {
	Name() string
	Validate() error
	Apply(ctx Context) error
	Remove(ctx Context) error
}

type Context struct {
	Executable string
	HomeDir    string
	Options    Options
}

func Run(agent string, opts Options) error {
	ctx, err := newContext()
	if err != nil {
		return err
	}
	ctx.Options = opts

	adapter, err := resolveAdapter(agent)
	if err != nil {
		return err
	}

	if err := adapter.Validate(); err != nil {
		return err
	}

	if opts.Uninstall {
		return adapter.Remove(ctx)
	}

	return adapter.Apply(ctx)
}

func SupportedAgents() []string {
	return []string{"opencode", "claude", "codex"}
}

func resolveAdapter(agent string) (Adapter, error) {
	switch strings.ToLower(strings.TrimSpace(agent)) {
	case "opencode":
		return OpenCodeAdapter{}, nil
	case "claude", "claude-code":
		return ClaudeCodeAdapter{}, nil
	case "codex":
		return CodexAdapter{}, nil
	case "gemini":
		return nil, fmt.Errorf("agent %q is not validated for setup in this phase yet", agent)
	default:
		return nil, fmt.Errorf("unsupported agent: %s", agent)
	}
}

func newContext() (Context, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Context{}, err
	}

	executable, err := resolvedExecutablePath()
	if err != nil {
		return Context{}, err
	}

	return Context{
		Executable: executable,
		HomeDir:    home,
	}, nil
}

func resolvedExecutablePath() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(executable, os.TempDir()) {
		return executable, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	localBinary := filepath.Join(wd, "vectos")
	if _, err := os.Stat(localBinary); err == nil {
		return localBinary, nil
	}

	return executable, nil
}
