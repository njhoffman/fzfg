package fzfg

import (
	"testing"
)

func TestComputePercent(t *testing.T) {
	tests := []struct {
		name   string
		dim    int
		offset int
		factor int
		want   int
	}{
		{"wide terminal", 200, 7000, 11, 54},
		{"narrow terminal", 80, 7000, 11, 50}, // clamps to 50
		{"very wide terminal", 300, 7000, 11, 65},
		{"tall terminal", 50, 4000, 5, 50}, // clamps to 50
		{"medium height", 100, 4000, 5, 55},
		{"zero dimension", 0, 7000, 11, 50}, // fallback
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computePercent(tt.dim, tt.offset, tt.factor)
			if got != tt.want {
				t.Errorf("computePercent(%d, %d, %d) = %d, want %d",
					tt.dim, tt.offset, tt.factor, got, tt.want)
			}
		})
	}
}

func TestComputePreviewLayout_Horizontal(t *testing.T) {
	ti := TerminalInfo{Width: 200, Height: 50, AspectRatio: 4.0}
	cfg := DefaultLayoutConfig()

	layout := ComputePreviewLayout(ti, cfg)

	if layout.Direction != "right" {
		t.Errorf("expected direction 'right', got %q", layout.Direction)
	}
	if layout.Percent < 50 || layout.Percent > 80 {
		t.Errorf("percent %d out of range [50, 80]", layout.Percent)
	}
	if layout.Setting == "" {
		t.Error("expected non-empty Setting")
	}
}

func TestComputePreviewLayout_Vertical(t *testing.T) {
	ti := TerminalInfo{Width: 80, Height: 60, AspectRatio: 1.33}
	cfg := DefaultLayoutConfig()

	layout := ComputePreviewLayout(ti, cfg)

	if layout.Direction != "bottom" {
		t.Errorf("expected direction 'bottom', got %q", layout.Direction)
	}
}

func TestComputePreviewLayout_NoTerminal(t *testing.T) {
	ti := TerminalInfo{Width: 0, Height: 0}
	cfg := DefaultLayoutConfig()

	layout := ComputePreviewLayout(ti, cfg)

	if layout.Percent != 50 {
		t.Errorf("expected 50%% fallback, got %d%%", layout.Percent)
	}
	if layout.Direction != "right" {
		t.Errorf("expected fallback direction 'right', got %q", layout.Direction)
	}
}

func TestDefaultLayoutConfig(t *testing.T) {
	cfg := DefaultLayoutConfig()
	if cfg.VerticalThreshold != 2.0 {
		t.Errorf("VerticalThreshold = %f, want 2.0", cfg.VerticalThreshold)
	}
	if cfg.HorizontalPreviewLocation != "right" {
		t.Errorf("HorizontalPreviewLocation = %q, want 'right'", cfg.HorizontalPreviewLocation)
	}
}

func TestFormatTerminalInfo(t *testing.T) {
	ti := TerminalInfo{
		Width:       200,
		Height:      50,
		IsTTY:       true,
		InTmux:      true,
		TmuxPane:    "%1",
		TmuxPanes:   3,
		AspectRatio: 4.0,
	}

	m := FormatTerminalInfo(ti)

	if m["width"] != "200" {
		t.Errorf("width = %q, want '200'", m["width"])
	}
	if m["tmux_panes"] != "3" {
		t.Errorf("tmux_panes = %q, want '3'", m["tmux_panes"])
	}
}

func TestComputeTmuxLayout_NotInTmux(t *testing.T) {
	ti := TerminalInfo{InTmux: false}
	layout := ComputeTmuxLayout(ti, "85%", "75%", 2, 140)

	if layout.UseFzfTmux {
		t.Error("should not use fzf-tmux when not in tmux")
	}
	if layout.UsePopup {
		t.Error("should not use popup when not in tmux")
	}
}

func TestComputeTmuxLayout_PopupConditions(t *testing.T) {
	// Wide terminal triggers popup
	ti := TerminalInfo{InTmux: true, Width: 200, TmuxPanes: 1}
	layout := ComputeTmuxLayout(ti, "85%", "75%", 2, 140)

	if !layout.UsePopup {
		t.Error("expected popup for wide terminal (200 > 140)")
	}

	// Multiple panes triggers popup
	ti2 := TerminalInfo{InTmux: true, Width: 100, TmuxPanes: 3}
	layout2 := ComputeTmuxLayout(ti2, "85%", "75%", 2, 140)

	if !layout2.UsePopup {
		t.Error("expected popup for multiple panes (3 >= 2)")
	}

	// Narrow, single pane = no popup
	ti3 := TerminalInfo{InTmux: true, Width: 100, TmuxPanes: 1}
	layout3 := ComputeTmuxLayout(ti3, "85%", "75%", 2, 140)

	if layout3.UsePopup {
		t.Error("should not use popup for narrow single-pane terminal")
	}
}
