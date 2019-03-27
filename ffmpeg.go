package astiffmpeg

import (
	"bytes"
	"context"
	"os/exec"
	"strings"

	"os"
	"time"

	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

// FFMpeg represents an entity capable of running an FFMpeg binary
// https://ffmpeg.org/ffmpeg.html
type FFMpeg struct {
	binaryPath   string
	stdErrParser StdErrParser
}

// New creates a new FFMpeg
func New(c Configuration) *FFMpeg {
	return &FFMpeg{binaryPath: c.BinaryPath}
}

// SetStdErrParser sets the stderr parser
func (f *FFMpeg) SetStdErrParser(s StdErrParser) {
	f.stdErrParser = s
}

// Exec executes the binary with the specified options
// ffmpeg [global_options] {[input_file_options] -i input_url} ... {[output_file_options] output_url} ...
func (f *FFMpeg) Exec(ctx context.Context, g GlobalOptions, in []Input, out []Output) (err error) {
	// Create cmd
	var cmd = exec.CommandContext(ctx, f.binaryPath)
	cmd.Env = os.Environ()

	// Output is redirected in stderr only
	var bufErr = &bytes.Buffer{}
	cmd.Stderr = bufErr

	// Global options
	g.adaptCmd(cmd)

	// Parse stderr
	if f.stdErrParser != nil {
		t := time.NewTicker(f.stdErrParser.Period())
		defer t.Stop()
		go func() {
			for t := range t.C {
				f.stdErrParser.Process(t, bufErr)
			}
		}()
	}

	// Inputs
	for idx, i := range in {
		if err = i.adaptCmd(cmd); err != nil {
			err = errors.Wrapf(err, "astiffmpeg: adapting cmd for input #%d failed", idx)
			return
		}
	}

	// Outputs
	for idx, o := range out {
		if err = o.adaptCmd(cmd); err != nil {
			err = errors.Wrapf(err, "astiffmpeg: adapting cmd for output #%d failed", idx)
			return
		}
	}

	// Run cmd
	astilog.Debugf("Executing %s", strings.Join(cmd.Args, " "))
	if err = cmd.Run(); err != nil {
		err = errors.Wrapf(err, "astiffmpeg: running %s failed with stderr %s", strings.Join(cmd.Args, " "), bufErr.Bytes())
		return
	}
	return
}

// Exec executes the binary with the specified options
// ffmpeg [global_options] {[input_file_options] -i input_url} ... {[output_file_options] output_url} ...
func (f *FFMpeg) BuildCmd(ctx context.Context, g GlobalOptions, in []Input,
	complex ComplexFilterOptions, out []Output) (cmd *exec.Cmd, err error) {
	// Create cmd
	cmd = exec.CommandContext(ctx, f.binaryPath)
	cmd.Env = os.Environ()

	// Global options
	g.adaptCmd(cmd)

	// Inputs
	for idx, i := range in {
		if err = i.adaptCmd(cmd); err != nil {
			err = errors.Wrapf(err, "astiffmpeg: adapting cmd for input #%d failed", idx)
			return
		}
	}

	complex.adaptCmd(cmd)

	// Outputs
	for idx, o := range out {
		if err = o.adaptCmd(cmd); err != nil {
			err = errors.Wrapf(err, "astiffmpeg: adapting cmd for output #%d failed", idx)
			return
		}
	}

	return
}
