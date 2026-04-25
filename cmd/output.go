package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// OutputConfig controls formatting behavior
type OutputConfig struct {
	Colors bool
	Emoji  bool
	Quiet  bool
}

// Writer provides formatted output with configurable styling
type Writer struct {
	out    io.Writer
	config OutputConfig
	err    error // first error encountered
}

// NewWriter creates a new Writer with the given configuration
func NewWriter(out io.Writer, config OutputConfig) *Writer {
	return &Writer{
		out:    out,
		config: config,
	}
}

// Message represents a structured message with optional formatting
type Message struct {
	Text  string
	Color string
	Emoji string
	Bold  bool
}

// Write outputs a message according to the writer's configuration
func (w *Writer) Write(msg Message) *Writer {
	if w.err != nil || w.config.Quiet {
		return w
	}

	var output string

	// Add emoji if enabled
	if w.config.Emoji && msg.Emoji != "" {
		output = msg.Emoji + " "
	}

	// Add color/bold if enabled
	if w.config.Colors {
		if msg.Bold {
			output += "\033[1m"
		}
		if msg.Color != "" {
			output += msg.Color
		}
	}

	output += msg.Text

	// Close formatting if enabled
	if w.config.Colors && (msg.Bold || msg.Color != "") {
		output += "\033[0m"
	}

	_, w.err = fmt.Fprint(w.out, output)
	return w
}

// Printf is like Write but with format string
func (w *Writer) Printf(msg Message, args ...any) *Writer {
	newMsg := msg
	newMsg.Text = fmt.Sprintf(msg.Text, args...)
	return w.Write(newMsg)
}

// Writeln writes a message followed by a newline
func (w *Writer) Writeln(msg Message) *Writer {
	return w.Write(msg).WriteString("\n")
}

// WriteString outputs plain text (no formatting)
func (w *Writer) WriteString(text string) *Writer {
	if w.err != nil || w.config.Quiet {
		return w
	}
	_, w.err = fmt.Fprint(w.out, text)
	return w
}

// WritelnString outputs plain text followed by a newline
func (w *Writer) WritelnString(text string) *Writer {
	if w.err != nil || w.config.Quiet {
		return w
	}

	_, w.err = fmt.Fprintln(w.out, text)
	return w
}

// ANSI color codes
const (
	ColorRed          = "\033[31m"
	ColorYellow       = "\033[33m"
	ColorCyan         = "\033[36m"
	ColorGray         = "\033[90m"
	ColorBrightGreen  = "\033[1;32m"
	ColorBrightYellow = "\033[1;33m"
	ColorBrightRed    = "\033[1;31m"
)

// Predefined message constructors for common patterns

func Success(text string) Message {
	return Message{Text: text, Color: ColorBrightGreen, Emoji: "✅", Bold: true}
}

func Error(text string) Message {
	return Message{Text: text, Emoji: "❌"}
}

func Warning(text string) Message {
	return Message{Text: text, Color: ColorBrightYellow, Emoji: "⚠️", Bold: true}
}

func Info(text string) Message {
	return Message{Text: text, Color: ColorYellow, Emoji: "💡"}
}

func Target(text string) Message {
	return Message{Text: text, Emoji: "🎯", Bold: true}
}

func Rocket(text string) Message {
	return Message{Text: text, Emoji: "🚀", Bold: true}
}

func Sparkles(text string) Message {
	return Message{Text: text, Emoji: "✨", Bold: true}
}

func Link(text string) Message {
	return Message{Text: text, Color: ColorCyan, Emoji: "🔗"}
}

func Plain(text string) Message {
	return Message{Text: text}
}

func Bold(text string) Message {
	return Message{Text: text, Bold: true}
}

func Colored(text, color string) Message {
	return Message{Text: text, Color: color}
}

// Global output configuration
var (
	globalConfig = OutputConfig{
		Colors: true, // auto-detect on first use
		Emoji:  true,
	}
	autoDetected bool
)

// SetGlobalConfig updates the global output configuration
func SetGlobalConfig(colors string, emoji, quiet bool) error {
	switch colors {
	case "auto":
		globalConfig.Colors = isTerminal()
	case "always":
		globalConfig.Colors = true
	case "never":
		globalConfig.Colors = false
	default:
		return fmt.Errorf("invalid color mode: %s (valid: auto, always, never)", colors)
	}

	// Check NO_COLOR environment variable (explicit flag takes precedence)
	if os.Getenv("NO_COLOR") != "" && colors == "auto" {
		globalConfig.Colors = false
	}

	globalConfig.Emoji = emoji
	globalConfig.Quiet = quiet
	autoDetected = true
	return nil
}

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// IsTerminal reports whether the writer's underlying output is a terminal.
func (w *Writer) IsTerminal() bool {
	f, ok := w.out.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

// autoDetectConfig performs one-time auto-detection if not explicitly configured
func autoDetectConfig() {
	if !autoDetected {
		if os.Getenv("NO_COLOR") != "" {
			globalConfig.Colors = false
		} else {
			globalConfig.Colors = isTerminal()
		}
		autoDetected = true
	}
}

// GetWriter returns a writer for the given cobra command
func GetWriter(cmd *cobra.Command) *Writer {
	autoDetectConfig()
	return NewWriter(cmd.OutOrStdout(), globalConfig)
}

// GetErrorWriter returns a writer for stderr
func GetErrorWriter() *Writer {
	autoDetectConfig()
	return NewWriter(os.Stderr, globalConfig)
}

// Err returns the first error encountered during writing
func (w *Writer) Err() error {
	return w.err
}
