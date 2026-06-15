package youtube

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
)

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

// parseSRT parses SubRip (SRT) caption data into TranscriptLines.
//
// SRT block format:
//
//	{index}
//	{HH:MM:SS,mmm} --> {HH:MM:SS,mmm}
//	{text line(s)}
//	{blank line}
func parseSRT(data []byte) ([]TranscriptLine, error) {
	raw := strings.ReplaceAll(string(data), "\r\n", "\n")
	blocks := strings.Split(strings.TrimSpace(raw), "\n\n")

	var lines []TranscriptLine
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		parts := strings.SplitN(block, "\n", 3)
		if len(parts) < 3 {
			continue
		}
		// parts[0] = index (ignore)
		// parts[1] = timestamp line
		// parts[2] = text (may have internal newlines)
		start, duration, ok := parseSRTTimestampLine(parts[1])
		if !ok {
			continue
		}
		// Collapse multi-line text into a single line and strip HTML tags.
		text := htmlTagRe.ReplaceAllString(parts[2], "")
		text = strings.Join(strings.Fields(text), " ")
		if text == "" {
			continue
		}
		lines = append(lines, TranscriptLine{
			Start:    start,
			Duration: duration,
			Text:     text,
		})
	}
	return lines, nil
}

// parseSRTTimestampLine parses "HH:MM:SS,mmm --> HH:MM:SS,mmm" into start/duration in seconds.
func parseSRTTimestampLine(line string) (start, duration float64, ok bool) {
	parts := strings.Split(line, " --> ")
	if len(parts) != 2 {
		return 0, 0, false
	}
	start, ok = parseSRTTimecode(strings.TrimSpace(parts[0]))
	if !ok {
		return 0, 0, false
	}
	end, ok := parseSRTTimecode(strings.TrimSpace(parts[1]))
	if !ok {
		return 0, 0, false
	}
	return start, end - start, true
}

// parseSRTTimecode converts "HH:MM:SS,mmm" to seconds.
func parseSRTTimecode(s string) (float64, bool) {
	s = strings.Replace(s, ",", ".", 1)
	var h, m int
	var sec float64
	if _, err := fmt.Sscanf(s, "%d:%d:%f", &h, &m, &sec); err != nil {
		return 0, false
	}
	return float64(h*3600+m*60) + sec, true
}

// parseTimedText parses YouTube's srv1 timedtext XML format.
//
// Format: <transcript><text start="0.5" dur="2.3">line text</text>...</transcript>
func parseTimedText(data []byte) ([]TranscriptLine, error) {
	type textElem struct {
		Start    float64 `xml:"start,attr"`
		Duration float64 `xml:"dur,attr"`
		Text     string  `xml:",chardata"`
	}
	type transcript struct {
		Lines []textElem `xml:"text"`
	}

	var t transcript
	if err := xml.Unmarshal(data, &t); err != nil {
		return nil, err
	}

	lines := make([]TranscriptLine, 0, len(t.Lines))
	for _, l := range t.Lines {
		text := strings.TrimSpace(l.Text)
		if text == "" {
			continue
		}
		lines = append(lines, TranscriptLine{
			Start:    l.Start,
			Duration: l.Duration,
			Text:     text,
		})
	}
	return lines, nil
}
