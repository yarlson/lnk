package cmd

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	error2 "github.com/yarlson/lnk/internal/lnkerror"
)

func TestOutputConfig(t *testing.T) {
	tests := []struct {
		name           string
		colors         string
		emoji          bool
		expectError    bool
		expectedColors bool
		expectedEmoji  bool
	}{
		{
			name:           "auto mode",
			colors:         "auto",
			emoji:          true,
			expectError:    false,
			expectedColors: false, // TTY detection will return false in tests
			expectedEmoji:  true,
		},
		{
			name:           "always mode",
			colors:         "always",
			emoji:          false,
			expectError:    false,
			expectedColors: true,
			expectedEmoji:  false,
		},
		{
			name:           "never mode",
			colors:         "never",
			emoji:          true,
			expectError:    false,
			expectedColors: false,
			expectedEmoji:  true,
		},
		{
			name:        "invalid mode",
			colors:      "invalid",
			emoji:       true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear NO_COLOR for consistent testing
			_ = os.Unsetenv("NO_COLOR")

			err := SetGlobalConfig(tt.colors, tt.emoji)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				if globalConfig.Colors != tt.expectedColors {
					t.Errorf("expected colors %v, got %v", tt.expectedColors, globalConfig.Colors)
				}
				if globalConfig.Emoji != tt.expectedEmoji {
					t.Errorf("expected emoji %v, got %v", tt.expectedEmoji, globalConfig.Emoji)
				}
			}
		})
	}
}

func TestNOCOLOREnvironmentVariable(t *testing.T) {
	// Test NO_COLOR environment variable with auto mode
	_ = os.Setenv("NO_COLOR", "1")
	defer func() { _ = os.Unsetenv("NO_COLOR") }()

	err := SetGlobalConfig("auto", true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if globalConfig.Colors != false {
		t.Errorf("expected colors disabled when NO_COLOR is set, got %v", globalConfig.Colors)
	}
}

func TestWriterOutput(t *testing.T) {
	tests := []struct {
		name           string
		config         OutputConfig
		message        Message
		expectedOutput string
	}{
		{
			name:   "full formatting",
			config: OutputConfig{Colors: true, Emoji: true},
			message: Message{
				Text:  "test message",
				Color: ColorRed,
				Emoji: "✅",
				Bold:  true,
			},
			expectedOutput: "✅ \033[1m\033[31mtest message\033[0m",
		},
		{
			name:   "colors only",
			config: OutputConfig{Colors: true, Emoji: false},
			message: Message{
				Text:  "test message",
				Color: ColorRed,
				Emoji: "✅",
				Bold:  true,
			},
			expectedOutput: "\033[1m\033[31mtest message\033[0m",
		},
		{
			name:   "emoji only",
			config: OutputConfig{Colors: false, Emoji: true},
			message: Message{
				Text:  "test message",
				Color: ColorRed,
				Emoji: "✅",
				Bold:  true,
			},
			expectedOutput: "✅ test message",
		},
		{
			name:   "no formatting",
			config: OutputConfig{Colors: false, Emoji: false},
			message: Message{
				Text:  "test message",
				Color: ColorRed,
				Emoji: "✅",
				Bold:  true,
			},
			expectedOutput: "test message",
		},
		{
			name:           "plain message",
			config:         OutputConfig{Colors: true, Emoji: true},
			message:        Plain("plain text"),
			expectedOutput: "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer := NewWriter(&buf, tt.config)

			writer.Write(tt.message)
			if err := writer.Err(); err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if buf.String() != tt.expectedOutput {
				t.Errorf("expected %q, got %q", tt.expectedOutput, buf.String())
			}
		})
	}
}

func TestPredefinedMessages(t *testing.T) {
	tests := []struct {
		name    string
		creator func(string) Message
		text    string
	}{
		{"Success", Success, "operation succeeded"},
		{"Error", Error, "something failed"},
		{"Warning", Warning, "be careful"},
		{"Info", Info, "useful information"},
		{"Target", Target, "target reached"},
		{"Rocket", Rocket, "launching"},
		{"Sparkles", Sparkles, "amazing"},
		{"Link", Link, "connected"},
		{"Plain", Plain, "no formatting"},
		{"Bold", Bold, "emphasis"},
	}

	var buf bytes.Buffer
	writer := NewWriter(&buf, OutputConfig{Colors: true, Emoji: true})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			msg := tt.creator(tt.text)

			writer.Write(msg)
			if err := writer.Err(); err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			output := buf.String()
			if !strings.Contains(output, tt.text) {
				t.Errorf("output should contain text %q, got %q", tt.text, output)
			}
		})
	}
}

func TestStructuredErrors(t *testing.T) {
	tests := []struct {
		name        string
		err         *error2.Error
		config      OutputConfig
		contains    []string
		notContains []string
	}{
		{
			name: "structured error with full formatting",
			err: &error2.Error{
				Err:        errors.New("something went wrong"),
				Path:       "/some/path",
				Suggestion: "try this instead",
			},
			config:   OutputConfig{Colors: true, Emoji: true},
			contains: []string{"❌", "something went wrong", "/some/path", "💡", "try this instead"},
		},
		{
			name: "structured error without emojis",
			err: &error2.Error{
				Err:        errors.New("something went wrong"),
				Path:       "/some/path",
				Suggestion: "try this instead",
			},
			config:      OutputConfig{Colors: true, Emoji: false},
			contains:    []string{"something went wrong", "/some/path", "try this instead"},
			notContains: []string{"❌", "💡"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := NewWriter(&buf, tt.config)

			_ = w.Write(Error(tt.err.Err.Error()))
			if tt.err.Path != "" {
				_ = w.WriteString("\n   ")
				_ = w.Write(Colored(tt.err.Path, ColorRed))
			}
			if tt.err.Suggestion != "" {
				_ = w.WriteString("\n   ")
				_ = w.Write(Info(tt.err.Suggestion))
			}

			output := buf.String()
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("output should contain %q, got %q", expected, output)
				}
			}
			for _, notExpected := range tt.notContains {
				if strings.Contains(output, notExpected) {
					t.Errorf("output should not contain %q, got %q", notExpected, output)
				}
			}
		})
	}
}
