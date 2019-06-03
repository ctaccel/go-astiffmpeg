package astiffmpeg

import (
	"os/exec"
	"reflect"
	"strconv"

	"fmt"
	"strings"

	"math"

	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

// GlobalOptions represents global options
type GlobalOptions struct {
	Log       *LogOptions
	NoStats   bool
	Overwrite *bool
	// Dump full command line and console output to a file named program-YYYYMMDD-HHMMSS.log in the current directory.
	// This file can be useful for bug reports. It also implies -loglevel verbose.
	Report bool
}

func (o GlobalOptions) adaptCmd(cmd *exec.Cmd) {
	cmd.Args = append(cmd.Args, "-hide_banner")
	if o.Log != nil {
		o.Log.adaptCmd(cmd)
	}
	if o.Overwrite != nil {
		if *o.Overwrite {
			cmd.Args = append(cmd.Args, "-y")
		} else {
			cmd.Args = append(cmd.Args, "-n")
		}
	}
	if o.NoStats {
		cmd.Args = append(cmd.Args, "-nostats")
	}
	if o.Report {
		cmd.Args = append(cmd.Args, "-report")
	}
}

// Log levels
const (
	// Show everything, including debugging information.
	LogLevelDebug = "debug"
	// Show all errors, including ones which can be recovered from.
	LogLevelError = "error"
	// Only show fatal errors. These are errors after which the process absolutely cannot continue.
	LogLevelFatal = "fatal"
	// Show informative messages during processing. This is in addition to warnings and errors. This is the default
	// value.
	LogLevelInfo = "info"
	// Only show fatal errors which could lead the process to crash, such as an assertion failure. This is not
	// currently used for anything.
	LogLevelPanic = "panic"
	// Show nothing at all; be silent.
	LogLevelQuiet = "quiet"
	// Same as info, except more verbose.
	LogLevelTrace = "trace"
	// Same as info, except more verbose.
	LogLevelVerbose = "verbose"
	// Show all warnings and errors. Any message related to possibly incorrect or unexpected events will be shown.
	LogLevelWarning = "warning"
)

// LogOptions represents log options
type LogOptions struct {
	Color    *bool
	Level    string
	Repeated bool
}

func (o LogOptions) adaptCmd(cmd *exec.Cmd) {
	if o.Color != nil {
		if *o.Color {
			cmd.Env = append(cmd.Env, "AV_LOG_FORCE_COLOR=1")
		} else {
			cmd.Env = append(cmd.Env, "AV_LOG_FORCE_NOCOLOR=1")
		}
	}
	if len(o.Level) > 0 {
		var v string
		if o.Repeated {
			v = "repeat+"
		}
		v += o.Level
		cmd.Args = append(cmd.Args, "-loglevel", v)
	}
}

// Number represents a number which value can be shortened using string shortcuts
type Number struct {
	BinaryMultiple bool   // Result will be based on powers of 1024 instead of powers of 1000.
	ByteMultiple   bool   // Multiplies the value by 8.
	Prefix         string // K, M, G, ...
	Value          interface{}
}

func numberFromString(i string) (n Number, err error) {
	if strings.HasSuffix(i, "B") {
		n.ByteMultiple = true
		i = strings.TrimSuffix(i, "B")
	}
	if strings.HasSuffix(i, "i") {
		n.BinaryMultiple = true
		i = strings.TrimSuffix(i, "i")
	}
	if _, err := strconv.Atoi(string(i[len(i)-1])); err != nil {
		n.Prefix = string(i[len(i)-1])
		i = i[:len(i)-1]
	}
	n.Value, err = strconv.ParseFloat(i, 64)
	return
}

func (n Number) float64() (o float64) {
	// Get initial value
	switch n.Value.(type) {
	case float64:
		o = n.Value.(float64)
	case int:
		o = float64(n.Value.(int))
	default:
		// TODO Remove this log
		astilog.Debugf("astiffmpeg: unlisted number type %#v", n.Value)
		return
	}

	// Get byte multiplier
	if n.ByteMultiple {
		o *= 8
	}

	// Get binary multiplier
	var m = 1000.0
	if n.BinaryMultiple {
		m = 1024.0
	}

	// Get prefix
	switch strings.ToLower(n.Prefix) {
	case "k":
		o *= m
	case "m":
		o *= math.Pow(m, 2)
	case "g":
		o *= math.Pow(m, 3)
	case "t":
		o *= math.Pow(m, 4)
	case "p":
		o *= math.Pow(m, 5)
	}
	return
}

func (n Number) string() (o string) {
	switch n.Value.(type) {
	case float64:
		o = strconv.FormatFloat(n.Value.(float64), 'f', 3, 64)
		for o[len(o)-1] == '0' {
			o = o[:len(o)-1]
		}
		o = strings.TrimSuffix(o, ".")
	case int:
		o = strconv.Itoa(n.Value.(int))
	default:
		// TODO Remove this log
		astilog.Debugf("astiffmpeg: unlisted number type %#v", n.Value)
		return
	}
	o += n.Prefix
	if n.BinaryMultiple {
		o += "i"
	}
	if n.ByteMultiple {
		o += "B"
	}
	return
}

// Stream specifier types
const (
	StreamSpecifierTypeAudio                = "a"
	StreamSpecifierTypeSubtitle             = "s"
	StreamSpecifierTypeVideo                = "v"
	StreamSpecifierTypeVideoAndNotThumbnail = "V"
)

// StreamSpecifier represents a stream specifier
type StreamSpecifier struct {
	Index *int
	Name  string
	Type  string
}

func (s StreamSpecifier) string() (o string) {
	if len(s.Name) > 0 {
		return s.Name
	}
	if len(s.Type) > 0 {
		o = s.Type
	}
	if s.Index != nil {
		if len(s.Type) > 0 {
			o += ":"
		}
		o += strconv.Itoa(*s.Index)
	}
	return
}

// Input represents an input
type Input struct {
	Options *InputOptions
	Path    string
}

func (i Input) adaptCmd(cmd *exec.Cmd) (err error) {
	if i.Options != nil {
		if err = i.Options.adaptCmd(cmd); err != nil {
			err = errors.Wrap(err, "astiffmpeg: adapting cmd for options failed")
			return
		}
	}
	cmd.Args = append(cmd.Args, "-i", i.Path)
	return
}

// InputOptions represents input options
type InputOptions struct {
	Decoding *DecodingOptions
}

func (o InputOptions) adaptCmd(cmd *exec.Cmd) (err error) {
	if o.Decoding != nil {
		if err = o.Decoding.adaptCmd(cmd); err != nil {
			err = errors.Wrap(err, "astiffmpeg: adapting cmd for decoding options failed")
			return
		}
	}
	return
}

// Deinterlacing modes
const (
	DeinterlacingModeAdaptive = "adaptive"
	DeinterlacingModeBob      = "bob"
	DeinterlacingModeWeave    = "weave"
)

// DecodingOptions represents decoding options
type DecodingOptions struct {
	Codec                      *StreamOption
	DeinterlacingMode          string
	DropSecondField            *bool
	Duration                   string
	HardwareAcceleration       string
	HardwareAccelerationDevice *int
	Position                   string
}

func (o DecodingOptions) adaptCmd(cmd *exec.Cmd) (err error) {
	if len(o.HardwareAcceleration) > 0 {
		cmd.Args = append(cmd.Args, "-hwaccel", o.HardwareAcceleration)
		if o.HardwareAccelerationDevice != nil {
			cmd.Args = append(cmd.Args, "-hwaccel_device", strconv.Itoa(*o.HardwareAccelerationDevice))
		}
	}
	if len(o.DeinterlacingMode) > 0 {
		cmd.Args = append(cmd.Args, "-deint", o.DeinterlacingMode)
	}
	if len(o.Duration) > 0 {
		cmd.Args = append(cmd.Args, "-t", o.Duration)
	}
	if len(o.Position) > 0 {
		cmd.Args = append(cmd.Args, "-ss", o.Position)
	}
	if o.DropSecondField != nil {
		v := "0"
		if *o.DropSecondField {
			v = "1"
		}
		cmd.Args = append(cmd.Args, "-drop_second_field", v)
	}
	if o.Codec != nil {
		if err = o.Codec.adaptCmd(cmd, "-c", func(i interface{}) (string, error) {
			if v, ok := i.(string); ok {
				return v, nil
			}
			return "", errors.New("astiffmpeg: value should be a string")
		}); err != nil {
			err = errors.Wrap(err, "astiffmpeg: adapting cmd for -c option failed")
			return
		}
	}
	return
}

// Output represents an output
type Output struct {
	Options *OutputOptions
	Path    string
}

func (o Output) adaptCmd(cmd *exec.Cmd) (err error) {
	if o.Options != nil {
		if err = o.Options.adaptCmd(cmd); err != nil {
			err = errors.Wrap(err, "astiffmpeg: adapting cmd for output failed")
			return
		}
	}
	cmd.Args = append(cmd.Args, "-y", o.Path)
	return
}

// SteamOption represents an option that can be specific to a stream
type StreamOption struct {
	Stream *StreamSpecifier
	Value  interface{}
}

func (o StreamOption) adaptCmd(cmd *exec.Cmd, name string, fn func(i interface{}) (string, error)) error {
	f := name
	if o.Stream != nil {
		f += ":" + o.Stream.string()
	}
	v, err := fn(o.Value)
	if err != nil {
		return errors.Wrapf(err, "astiffmpeg: adapting cmd for stream option %s failed", name)
	}
	cmd.Args = append(cmd.Args, f, v)
	return nil
}

// Coders
const (
	CoderAC      = "ac"
	CoderCABAC   = "cabac"
	CoderCAVLC   = "cavlc"
	CoderDefault = "default"
	CoderVLC     = "vlc"
)

// Presets
const (
	PresetUltrafast = "ultrafast"
	PresetSuperfast = "superfast"
	PresetVeryfast  = "veryfast"
	PresetFaster    = "faster"
	PresetFast      = "fast"
	PresetMedium    = "medium"
	PresetSlow      = "slow"
	PresetSlower    = "slower"
	PresetVeryslow  = "veryslow"
)

// Profiles
const (
	ProfileBaseline = "baseline"
	ProfileHigh     = "high"
	ProfileHigh10   = "high10"
	ProfileHigh422  = "high422"
	ProfileHigh444  = "high444"
	ProfileMain     = "main"
)

// Rate controls
const ()

// Tunes
const (
	TuneAnimation   = "animation"
	TuneFastdecode  = "fastdecode"
	TuneFilm        = "film"
	TuneGrain       = "grain"
	TuneStillimage  = "stillimage"
	TuneZerolatency = "zerolatency"
)

// OutputOptions represents output options
type OutputOptions struct {
	Encoding *EncodingOptions
	Format   string
	Map      *MapOptions
}

func (o OutputOptions) adaptCmd(cmd *exec.Cmd) (err error) {
	if o.Map != nil {
		o.Map.adaptCmd(cmd)
	}
	if o.Encoding != nil {
		if err = o.Encoding.adaptCmd(cmd); err != nil {
			err = errors.Wrap(err, "astiffmpeg: adapting cmd for encoding options failed")
			return
		}
	}
	if len(o.Format) > 0 {
		cmd.Args = append(cmd.Args, "-f", o.Format)
	}
	return
}

type ComplexFilter struct {
	Filters       []string
	InputStreams  []StreamSpecifier
	OutputStreams []StreamSpecifier
}

// ComplexFilterOptions represents complex filter option
type ComplexFilterOptions struct {
	OutputNum      *int
	ComplexFilters []ComplexFilter
}

func (o ComplexFilterOptions) adaptCmd(cmd *exec.Cmd) (err error) {
	if o.OutputNum != nil {
		s := fmt.Sprintf("split=%d", *o.OutputNum)
		for index := 0; index < *o.OutputNum; index++ {
			s = s + fmt.Sprintf("[out%d]", index)
		}
		o.ComplexFilters = append(o.ComplexFilters, ComplexFilter{
			Filters: []string{s},
		})
	}

	if len(o.ComplexFilters) > 0 {
		var vs []string
		for _, cf := range o.ComplexFilters {
			var v string
			for _, i := range cf.InputStreams {
				v += "[" + i.string() + "]"
			}
			v += strings.Join(cf.Filters, ",")
			for _, o := range cf.OutputStreams {
				v += "[" + o.string() + "]"
			}
			if len(v) != 0 {
				vs = append(vs, v)
			}
		}
		// cmd.Args = append(cmd.Args, "-filter_complex", "\""+strings.Join(vs, ";")+"\"")
		cmd.Args = append(cmd.Args, "-filter_complex", strings.Join(vs, ","))

	}

	return
}

// EncodingOptions represents encoding options
type EncodingOptions struct {
	AudioSamplerate    *int
	AudioChannels      *int
	BFrames            *int
	Bitrate            []StreamOption
	BStrategy          *int
	BufSize            *Number
	Codec              []StreamOption
	Coder              string
	ConstantQuality    *float64
	CRF                *int
	Filters            []StreamOption
	Framerate          *float64
	FrameSize          string
	GOP                *int
	KeyintMin          *int
	Level              *float64
	Maxrate            []StreamOption
	Minrate            []StreamOption
	Preset             string
	Profile            []StreamOption
	RateControl        string
	SCThreshold        *int
	Tune               string
	MaxMuxingQSize     *int
	Customize          map[string]interface{} // the third party, e.g IDT
	RemoveAudio        string
	HlsTime            *int
	HlsListSize        *int
	HlsKeyInfoFile     string
	HlsSegmentFileName string
}

func (o EncodingOptions) adaptCmd(cmd *exec.Cmd) (err error) {
	if o.AudioSamplerate != nil {
		cmd.Args = append(cmd.Args, "-ar", strconv.Itoa(*o.AudioSamplerate))
	}
	if o.AudioChannels != nil {
		cmd.Args = append(cmd.Args, "-ac", strconv.Itoa(*o.AudioChannels))
	}
	if o.BFrames != nil {
		cmd.Args = append(cmd.Args, "-bf", strconv.Itoa(*o.BFrames))
	}

	for idx, ro := range o.Bitrate {
		if err = ro.adaptCmd(cmd, "-b", func(i interface{}) (string, error) {
			if v, ok := i.(Number); ok {
				return v.string(), nil
			}
			return "", errors.New("astiffmpeg: value should be a Number")
		}); err != nil {
			err = errors.Wrapf(err, "astiffmpeg: adapting cmd for -b option #%d failed", idx)
			return
		}
	}
	if o.BStrategy != nil {
		cmd.Args = append(cmd.Args, "-b_strategy", strconv.Itoa(*o.BStrategy))
	}
	if o.BufSize != nil {
		cmd.Args = append(cmd.Args, "-bufsize", o.BufSize.string())
	}

	for idx, ro := range o.Codec {
		if err = ro.adaptCmd(cmd, "-codec", func(i interface{}) (string, error) {
			if v, ok := i.(string); ok {
				return v, nil
			}
			return "", errors.New("astiffmpeg: value should be a string")
		}); err != nil {
			err = errors.Wrapf(err, "astiffmpeg: adapting cmd for -codec option #%d failed", idx)
			return
		}
	}
	if len(o.Coder) > 0 {
		cmd.Args = append(cmd.Args, "-coder", o.Coder)
	}
	if o.ConstantQuality != nil {
		cmd.Args = append(cmd.Args, "-cq", strconv.FormatFloat(*o.ConstantQuality, 'f', 3, 64))
	}
	if o.CRF != nil {
		cmd.Args = append(cmd.Args, "-crf", strconv.Itoa(*o.CRF))
	}
	for idx, ro := range o.Filters {
		if err = ro.adaptCmd(cmd, "-filter", func(i interface{}) (string, error) {
			if v, ok := i.(FilterOptions); ok {
				return v.string(), nil
			}
			return "", errors.New("astiffmpeg: value should be a FilterOptions")
		}); err != nil {
			err = errors.Wrapf(err, "astiffmpeg: adapting cmd for -filter option #%d failed", idx)
			return
		}
	}
	if o.Framerate != nil {
		cmd.Args = append(cmd.Args, "-r", strconv.FormatFloat(*o.Framerate, 'f', 3, 64))
	}
	if len(o.FrameSize) > 0 {
		cmd.Args = append(cmd.Args, "-s", o.FrameSize)
	}
	if o.GOP != nil {
		cmd.Args = append(cmd.Args, "-g", strconv.Itoa(*o.GOP))
	}
	if o.KeyintMin != nil {
		cmd.Args = append(cmd.Args, "-keyint_min", strconv.Itoa(*o.KeyintMin))
	}
	if o.Level != nil {
		cmd.Args = append(cmd.Args, "-level", strconv.FormatFloat(*o.Level, 'f', 1, 64))
	}
	for idx, ro := range o.Maxrate {
		if err = ro.adaptCmd(cmd, "-maxrate", func(i interface{}) (string, error) {
			if v, ok := i.(Number); ok {
				return v.string(), nil
			}
			return "", errors.New("astiffmpeg: value should be a Number")
		}); err != nil {
			err = errors.Wrapf(err, "astiffmpeg: adapting cmd for -maxrate option #%d failed", idx)
			return
		}
	}
	for idx, ro := range o.Minrate {
		if err = ro.adaptCmd(cmd, "-minrate", func(i interface{}) (string, error) {
			if v, ok := i.(Number); ok {
				return v.string(), nil
			}
			return "", errors.New("astiffmpeg: value should be a Number")
		}); err != nil {
			err = errors.Wrapf(err, "astiffmpeg: adapting cmd for -minrate option #%d failed", idx)
			return
		}
	}
	if len(o.Preset) > 0 {
		cmd.Args = append(cmd.Args, "-preset", o.Preset)
	}
	for idx, ro := range o.Profile {
		if err = ro.adaptCmd(cmd, "-profile", func(i interface{}) (string, error) {
			if v, ok := i.(string); ok {
				return v, nil
			}
			return "", errors.New("astiffmpeg: value should be a string")
		}); err != nil {
			err = errors.Wrapf(err, "astiffmpeg: adapting cmd for -profile option #%d failed", idx)
			return
		}
	}
	if len(o.RateControl) > 0 {
		cmd.Args = append(cmd.Args, "-rc", o.RateControl)
	}
	if o.SCThreshold != nil {
		cmd.Args = append(cmd.Args, "-sc_threshold", strconv.Itoa(*o.SCThreshold))
	}
	if len(o.Tune) > 0 {
		cmd.Args = append(cmd.Args, "-tune", o.Tune)
	}
	if o.MaxMuxingQSize != nil {
		cmd.Args = append(cmd.Args, "-max_muxing_queue_size", strconv.Itoa(*o.MaxMuxingQSize))
	}
	if o.HlsTime != nil {
		cmd.Args = append(cmd.Args, "-hls_time", strconv.Itoa(*o.HlsTime))
	}
	if o.HlsListSize != nil {
		cmd.Args = append(cmd.Args, "-hls_list_size", strconv.Itoa(*o.HlsListSize))
	}
	if len(o.HlsKeyInfoFile) > 0 {
		cmd.Args = append(cmd.Args, "-hls_key_info_file", o.HlsKeyInfoFile)
	}
	if len(o.HlsSegmentFileName) > 0 {
		cmd.Args = append(cmd.Args, "-hls_segment_filename", o.HlsSegmentFileName)
	}
	for key, value := range o.Customize {
		t := reflect.TypeOf(value)
		switch t.Kind() {
		case reflect.Int:
			cmd.Args = append(cmd.Args, fmt.Sprintf("-%s", key), strconv.Itoa(int(value.(int))))
		case reflect.Float64:
			cmd.Args = append(cmd.Args, fmt.Sprintf("-%s", key), strconv.Itoa(int(value.(float64))))
		default:
			cmd.Args = append(cmd.Args, fmt.Sprintf("-%s", key), value.(string))
		}
	}
	if o.RemoveAudio == "y" {
		cmd.Args = append(cmd.Args, "-an")
	}
	return
}

// Ratio represents a ration
type Ratio struct {
	Antecedent, Consequent int
}

func (r Ratio) string() string {
	return fmt.Sprintf("%d/%d", r.Antecedent, r.Consequent)
}

// Scale represents a scale
type Scale struct {
	Width, Height int
}

func (s Scale) string() string {
	return fmt.Sprintf("%d:%d", s.Width, s.Height)
}

// FilterOptions represents filter options
type FilterOptions struct {
	SAR      *Ratio
	ScaleNPP *Scale
}

func (o FilterOptions) add(k, v string) string {
	return fmt.Sprintf("%s=%s", k, v)
}

func (o FilterOptions) string() string {
	var items []string
	if o.SAR != nil {
		items = append(items, o.add("setsar", o.SAR.string()))
	}
	if o.ScaleNPP != nil {
		items = append(items, o.add("scale_npp", o.ScaleNPP.string()))
	}
	return strings.Join(items, ",")
}

// MapOptions represents a set of map options
type MapOptions []MapOption

func (os MapOptions) adaptCmd(cmd *exec.Cmd) {
	for _, o := range os {
		o.adaptCmd(cmd)
	}
}

// MapOption represents a map option
type MapOption struct {
	Name        string
	InputFileID int
	Stream      *StreamSpecifier
}

func (o MapOption) adaptCmd(cmd *exec.Cmd) {
	if len(o.Name) > 0 {
		cmd.Args = append(cmd.Args, "-map", o.Name)
		return
	}
	v := strconv.Itoa(o.InputFileID)
	if o.Stream != nil {
		v += ":" + o.Stream.string()
	}
	cmd.Args = append(cmd.Args, "-map", v)
}
