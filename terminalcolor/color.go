package terminalcolor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mattn/go-colorable"
)

type LogWithColorAndStyle struct {
	Text   []string
	Color  []string
	Style  []string
	Format string
}

func (l LogWithColorAndStyle) Print() {
	var output string

	if l.Format == "" {
		output = strings.Join(l.Text, "")
	} else {
		defer func() {
			if r := recover(); r != nil {
				output = strings.Join(l.Text, "")
			}
		}()
		output = l.Format
		for i := range l.Text {
			placeholder := fmt.Sprintf("{text[%d]}", i)
			output = strings.ReplaceAll(output, placeholder, "%s")
		}
		output = fmt.Sprintf(output, toInterfaceSlice(l.Text)...)
	}

	coloredOutput := applyColorAndStyle(output, l.Color, l.Style, l.Text)
	fmt.Fprintln(colorable.NewColorableStdout(), coloredOutput)
}

func toInterfaceSlice(slice []string) []interface{} {
	ifaceSlice := make([]interface{}, len(slice))
	for i, v := range slice {
		ifaceSlice[i] = v
	}
	return ifaceSlice
}

func applyColorAndStyle(text string, colors []string, styles []string, texts []string) string {
	colorCodes := map[string]string{
		"red":     "\033[31m",
		"green":   "\033[32m",
		"yellow":  "\033[33m",
		"blue":    "\033[34m",
		"reset":   "\033[0m",
		"info":    "\033[36m",
		"warning": "\033[33m",
		"error":   "\033[31m",
	}

	styleCodes := map[string]string{
		"bold":           "\033[1m",
		"underline":      "\033[4m",
		"bold&underline": "\033[1;4m",
	}

	// Apply the first color to the entire text
	if len(colors) > 0 {
		wholeColorCode := getColorCode(colors[0], colorCodes)
		text = fmt.Sprintf("%s%s%s", wholeColorCode, text, colorCodes["reset"])
	}

	// Apply subsequent colors and styles to each text segment
	for i, segment := range texts {
		colorCode := ""
		styleCode := ""

		if i+1 < len(colors) {
			colorCode = getColorCode(colors[i+1], colorCodes)
		}

		if i+1 < len(styles) {
			styleCode = getStyleCode(styles[i+1], styleCodes)
		}

		coloredSegment := fmt.Sprintf("%s%s%s%s", colorCode, styleCode, segment, colorCodes["reset"])
		text = strings.Replace(text, segment, coloredSegment, 1)
	}

	return text
}

func getColorCode(color string, colorCodes map[string]string) string {
	if code, exists := colorCodes[color]; exists {
		return code
	} else if strings.HasPrefix(color, "#") && len(color) == 7 {
		r, _ := strconv.ParseInt(color[1:3], 16, 64)
		g, _ := strconv.ParseInt(color[3:5], 16, 64)
		b, _ := strconv.ParseInt(color[5:7], 16, 64)
		return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
	}
	return colorCodes["reset"]
}

func getStyleCode(style string, styleCodes map[string]string) string {
	if code, exists := styleCodes[style]; exists {
		return code
	}
	return ""
}
