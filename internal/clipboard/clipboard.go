package clipboard

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func WriteString(s string) error {
	return Write([]byte(s))
}

func WriteStringContext(ctx context.Context, s string) error {
	return WriteContext(ctx, []byte(s))
}

func Write(b []byte) error {
	return WriteContext(context.Background(), b)
}

func WriteContext(ctx context.Context, b []byte) error {
	var cmd *exec.Cmd

	switch {
	case runtime.GOOS == "darwin":
		cmd = exec.CommandContext(ctx, "pbcopy")
	case runtime.GOOS == "linux":
		if os.Getenv("WAYLAND_DISPLAY") != "" {
			cmd = exec.CommandContext(ctx, "wl-copy", "-t", "text/plain")
		} else {
			cmd = exec.CommandContext(ctx, "xclip", "-in", "-selection", "clipboard")
		}
	default:
		return fmt.Errorf("OS %s not supported", runtime.GOOS)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	_, err = stdin.Write(b)
	if err != nil {
		return err
	}

	err = stdin.Close()
	if err != nil {
		return err
	}

	return cmd.Wait()
}
