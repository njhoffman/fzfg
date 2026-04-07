package fzfg

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/term"
)

// LayoutConfig holds configurable thresholds for auto-sizing.
type LayoutConfig struct {
	VerticalThreshold              float64 // width/height ratio below which layout is vertical (default: 2.0)
	VerticalPreviewLocation        string  // "bottom" or "top"
	HorizontalPreviewLocation      string  // "right" or "left"
	HorizontalPreviewPercentParams [2]int  // [offset, factor] for: max(50, min(80, 100 - ((offset + (factor * W)) / W)))
	VerticalPreviewPercentParams   [2]int  // [offset, factor] for: max(50, min(80, 100 - ((offset + (factor * H)) / H)))
}

// TerminalInfo holds detected terminal and tmux state.
type TerminalInfo struct {
	Width       int
	Height      int
	IsTTY       bool
	InTmux      bool
	TmuxPane    string
	TmuxPanes   int // number of panes in current window
	AspectRatio float64
}

// TmuxLayout holds tmux-specific layout decisions.
type TmuxLayout struct {
	UseFzfTmux  bool   // whether to use fzf-tmux wrapper
	UsePopup    bool   // use popup instead of split
	PopupWidth  string // e.g. "85%"
	PopupHeight string // e.g. "75%"
	SplitLayout string // e.g. "-d 50%"
}

// PreviewLayout holds the computed preview window settings.
type PreviewLayout struct {
	Direction string // "right", "left", "top", "bottom"
	Percent   int    // size percentage
	Setting   string // formatted fzf --preview-window value e.g. "right:65%"
}

// DefaultLayoutConfig returns the default auto-sizing configuration,
// matching the defaults from auto-sized-fzf.
func DefaultLayoutConfig() LayoutConfig {
	return LayoutConfig{
		VerticalThreshold:              2.0,
		VerticalPreviewLocation:        "bottom",
		HorizontalPreviewLocation:      "right",
		HorizontalPreviewPercentParams: [2]int{7000, 11},
		VerticalPreviewPercentParams:   [2]int{4000, 5},
	}
}

// DetectTerminal gathers terminal dimensions and tmux state.
func DetectTerminal() TerminalInfo {
	info := TerminalInfo{}

	fd := os.Stdout.Fd()
	info.IsTTY = term.IsTerminal(fd)

	if w, h, err := term.GetSize(fd); err == nil {
		info.Width = w
		info.Height = h
	}

	// Fallback: try stderr if stdout is piped
	if info.Width == 0 || info.Height == 0 {
		if w, h, err := term.GetSize(os.Stderr.Fd()); err == nil {
			info.Width = w
			info.Height = h
		}
	}

	if info.Height > 0 {
		info.AspectRatio = float64(info.Width) / float64(info.Height)
	}

	// Tmux detection
	info.InTmux = os.Getenv("TMUX") != ""
	info.TmuxPane = os.Getenv("TMUX_PANE")

	if info.InTmux {
		info.TmuxPanes = detectTmuxPaneCount()
	}

	return info
}

// detectTmuxPaneCount runs `tmux list-panes` and counts the result lines.
func detectTmuxPaneCount() int {
	out, err := exec.Command("tmux", "list-panes").Output()
	if err != nil {
		return 1
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return len(lines)
}

// ComputeTmuxLayout determines whether to use fzf-tmux, popup, or split mode
// based on the layout config from fzfg.yaml and terminal state.
func ComputeTmuxLayout(ti TerminalInfo, popupWidthPct, popupHeightPct string, popupPaneThreshold, popupColsThreshold int) TmuxLayout {
	layout := TmuxLayout{
		PopupWidth:  popupWidthPct,
		PopupHeight: popupHeightPct,
		SplitLayout: "-d 50%",
	}

	if !ti.InTmux {
		return layout
	}

	// fzf-tmux is available if we're in tmux
	if _, err := exec.LookPath("fzf-tmux"); err == nil {
		layout.UseFzfTmux = true
	}

	// Popup if 2+ panes or wide terminal
	if ti.TmuxPanes >= popupPaneThreshold || ti.Width > popupColsThreshold {
		layout.UsePopup = true
	}

	return layout
}

// ComputePreviewLayout calculates the optimal preview window position and size
// based on terminal dimensions, porting the auto-sized-fzf algorithm.
//
// Algorithm:
//   - If width/height < threshold -> vertical layout (preview on bottom)
//   - Otherwise -> horizontal layout (preview on right)
//   - Size: max(50, min(80, 100 - ((offset + (factor * dimension)) / dimension)))
func ComputePreviewLayout(ti TerminalInfo, cfg LayoutConfig) PreviewLayout {
	layout := PreviewLayout{}

	// Default fallback if no terminal info
	if ti.Width == 0 || ti.Height == 0 {
		layout.Direction = cfg.HorizontalPreviewLocation
		layout.Percent = 50
		layout.Setting = fmt.Sprintf("%s:%d%%", layout.Direction, layout.Percent)
		return layout
	}

	isVertical := ti.AspectRatio < cfg.VerticalThreshold

	if isVertical {
		layout.Direction = cfg.VerticalPreviewLocation
		layout.Percent = computePercent(
			ti.Height,
			cfg.VerticalPreviewPercentParams[0],
			cfg.VerticalPreviewPercentParams[1],
		)
	} else {
		layout.Direction = cfg.HorizontalPreviewLocation
		layout.Percent = computePercent(
			ti.Width,
			cfg.HorizontalPreviewPercentParams[0],
			cfg.HorizontalPreviewPercentParams[1],
		)
	}

	layout.Setting = fmt.Sprintf("%s:%d%%", layout.Direction, layout.Percent)
	return layout
}

// computePercent implements: max(50, min(80, 100 - ((offset + (factor * dim)) / dim)))
func computePercent(dim, offset, factor int) int {
	if dim == 0 {
		return 50
	}
	raw := 100.0 - float64(offset+factor*dim)/float64(dim)
	clamped := math.Max(50, math.Min(80, raw))
	return int(clamped)
}

// FormatTerminalInfo returns a map of key-value pairs for debug/snapshot output.
func FormatTerminalInfo(ti TerminalInfo) map[string]string {
	m := map[string]string{
		"width":        strconv.Itoa(ti.Width),
		"height":       strconv.Itoa(ti.Height),
		"aspect_ratio": fmt.Sprintf("%.2f", ti.AspectRatio),
		"is_tty":       strconv.FormatBool(ti.IsTTY),
		"in_tmux":      strconv.FormatBool(ti.InTmux),
	}
	if ti.InTmux {
		m["tmux_pane"] = ti.TmuxPane
		m["tmux_panes"] = strconv.Itoa(ti.TmuxPanes)
	}
	return m
}
