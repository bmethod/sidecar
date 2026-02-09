package mouse

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRect_Contains(t *testing.T) {
	r := Rect{X: 10, Y: 20, W: 30, H: 40}

	tests := []struct {
		name   string
		x, y   int
		expect bool
	}{
		{"inside", 15, 30, true},
		{"top-left corner", 10, 20, true},
		{"right edge exclusive", 40, 30, false},
		{"bottom edge exclusive", 15, 60, false},
		{"just inside right", 39, 30, true},
		{"just inside bottom", 15, 59, true},
		{"left of rect", 9, 30, false},
		{"above rect", 15, 19, false},
		{"far outside", 100, 100, false},
		{"negative coords inside", 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.Contains(tt.x, tt.y); got != tt.expect {
				t.Errorf("Rect%+v.Contains(%d, %d) = %v, want %v", r, tt.x, tt.y, got, tt.expect)
			}
		})
	}
}

func TestRect_Contains_ZeroSize(t *testing.T) {
	zeroW := Rect{X: 5, Y: 5, W: 0, H: 10}
	if zeroW.Contains(5, 5) {
		t.Error("zero-width rect should not contain any point")
	}

	zeroH := Rect{X: 5, Y: 5, W: 10, H: 0}
	if zeroH.Contains(5, 5) {
		t.Error("zero-height rect should not contain any point")
	}

	zeroBoth := Rect{X: 5, Y: 5, W: 0, H: 0}
	if zeroBoth.Contains(5, 5) {
		t.Error("zero-size rect should not contain any point")
	}
}

func TestHitMap_AddAndTest(t *testing.T) {
	hm := NewHitMap()
	hm.Add("a", Rect{X: 0, Y: 0, W: 10, H: 10}, "data-a")
	hm.Add("b", Rect{X: 20, Y: 20, W: 10, H: 10}, "data-b")

	r := hm.Test(5, 5)
	if r == nil || r.ID != "a" {
		t.Fatalf("expected region 'a', got %v", r)
	}
	if r.Data != "data-a" {
		t.Errorf("expected data 'data-a', got %v", r.Data)
	}

	r = hm.Test(25, 25)
	if r == nil || r.ID != "b" {
		t.Fatalf("expected region 'b', got %v", r)
	}
}

func TestHitMap_OverlappingRegions(t *testing.T) {
	hm := NewHitMap()
	hm.Add("bottom", Rect{X: 0, Y: 0, W: 20, H: 20}, "bottom-data")
	hm.Add("top", Rect{X: 5, Y: 5, W: 10, H: 10}, "top-data")

	r := hm.Test(7, 7)
	if r == nil || r.ID != "top" {
		t.Fatalf("overlapping point should hit 'top' (last added), got %v", r)
	}

	r = hm.Test(2, 2)
	if r == nil || r.ID != "bottom" {
		t.Fatalf("non-overlapping point should hit 'bottom', got %v", r)
	}
}

func TestHitMap_Clear(t *testing.T) {
	hm := NewHitMap()
	hm.Add("a", Rect{X: 0, Y: 0, W: 10, H: 10}, nil)

	if hm.Test(5, 5) == nil {
		t.Fatal("expected hit before clear")
	}

	hm.Clear()

	if hm.Test(5, 5) != nil {
		t.Fatal("expected nil after clear")
	}
}

func TestHitMap_AddRect(t *testing.T) {
	hm := NewHitMap()
	hm.AddRect("r", 10, 20, 30, 40, "rect-data")

	r := hm.Test(15, 30)
	if r == nil || r.ID != "r" {
		t.Fatalf("expected region 'r', got %v", r)
	}
	if r.Rect.X != 10 || r.Rect.Y != 20 || r.Rect.W != 30 || r.Rect.H != 40 {
		t.Errorf("unexpected rect values: %+v", r.Rect)
	}
	if r.Data != "rect-data" {
		t.Errorf("expected data 'rect-data', got %v", r.Data)
	}
}

func TestHitMap_Regions(t *testing.T) {
	hm := NewHitMap()
	hm.Add("a", Rect{X: 0, Y: 0, W: 10, H: 10}, nil)
	hm.Add("b", Rect{X: 20, Y: 20, W: 10, H: 10}, nil)

	regions := hm.Regions()
	if len(regions) != 2 {
		t.Fatalf("expected 2 regions, got %d", len(regions))
	}

	regions[0].ID = "mutated"
	if hm.Regions()[0].ID == "mutated" {
		t.Error("Regions() should return a copy, but mutation affected original")
	}
}

func TestHitMap_TestMiss(t *testing.T) {
	hm := NewHitMap()
	hm.Add("a", Rect{X: 0, Y: 0, W: 5, H: 5}, nil)

	if hm.Test(50, 50) != nil {
		t.Error("expected nil for point outside all regions")
	}

	empty := NewHitMap()
	if empty.Test(0, 0) != nil {
		t.Error("expected nil on empty hit map")
	}
}

func TestHandler_HandleClick(t *testing.T) {
	h := NewHandler()
	h.HitMap.Add("btn", Rect{X: 0, Y: 0, W: 10, H: 10}, "button")

	result := h.HandleClick(5, 5)
	if result.Region == nil {
		t.Fatal("expected a region hit")
	}
	if result.Region.ID != "btn" {
		t.Errorf("expected region 'btn', got %q", result.Region.ID)
	}
	if result.IsDoubleClick {
		t.Error("first click should not be a double click")
	}
}

func TestHandler_HandleClick_Miss(t *testing.T) {
	h := NewHandler()
	h.HitMap.Add("btn", Rect{X: 0, Y: 0, W: 10, H: 10}, nil)

	result := h.HandleClick(50, 50)
	if result.Region != nil {
		t.Error("expected nil region for miss")
	}
}

func TestHandler_DoubleClick(t *testing.T) {
	h := NewHandler()
	h.HitMap.Add("btn", Rect{X: 0, Y: 0, W: 10, H: 10}, nil)

	first := h.HandleClick(5, 5)
	if first.IsDoubleClick {
		t.Error("first click should not be double click")
	}

	second := h.HandleClick(5, 5)
	if !second.IsDoubleClick {
		t.Error("second immediate click on same region should be double click")
	}

	third := h.HandleClick(5, 5)
	if third.IsDoubleClick {
		t.Error("third click should not be double click (reset after double)")
	}
}

func TestHandler_DragLifecycle(t *testing.T) {
	h := NewHandler()

	if h.IsDragging() {
		t.Error("should not be dragging initially")
	}

	h.StartDrag(10, 20, "divider", 200)

	if !h.IsDragging() {
		t.Error("should be dragging after StartDrag")
	}
	if h.DragRegion() != "divider" {
		t.Errorf("expected drag region 'divider', got %q", h.DragRegion())
	}
	if h.DragStartValue() != 200 {
		t.Errorf("expected drag start value 200, got %d", h.DragStartValue())
	}

	dx, dy := h.DragDelta(15, 25)
	if dx != 5 || dy != 5 {
		t.Errorf("expected drag delta (5, 5), got (%d, %d)", dx, dy)
	}

	dx, dy = h.DragDelta(5, 10)
	if dx != -5 || dy != -10 {
		t.Errorf("expected drag delta (-5, -10), got (%d, %d)", dx, dy)
	}

	h.EndDrag()

	if h.IsDragging() {
		t.Error("should not be dragging after EndDrag")
	}
	if h.DragRegion() != "" {
		t.Errorf("drag region should be empty after EndDrag, got %q", h.DragRegion())
	}
}

func TestHandler_Clear(t *testing.T) {
	h := NewHandler()
	h.HitMap.Add("btn", Rect{X: 0, Y: 0, W: 10, H: 10}, nil)

	if h.HitMap.Test(5, 5) == nil {
		t.Fatal("expected hit before clear")
	}

	h.Clear()

	if h.HitMap.Test(5, 5) != nil {
		t.Error("expected no hit after clear")
	}
}

func TestHandleMouse_Click(t *testing.T) {
	h := NewHandler()
	h.HitMap.Add("btn", Rect{X: 0, Y: 0, W: 10, H: 10}, nil)

	action := h.HandleMouse(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      5,
		Y:      5,
	})
	if action.Type != ActionClick {
		t.Errorf("expected ActionClick, got %d", action.Type)
	}
	if action.Region == nil || action.Region.ID != "btn" {
		t.Error("expected region 'btn'")
	}
}

func TestHandleMouse_ClickMiss(t *testing.T) {
	h := NewHandler()
	h.HitMap.Add("btn", Rect{X: 0, Y: 0, W: 10, H: 10}, nil)

	action := h.HandleMouse(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      50,
		Y:      50,
	})
	if action.Type != ActionNone {
		t.Errorf("expected ActionNone for miss, got %d", action.Type)
	}
}

func TestHandleMouse_DoubleClick(t *testing.T) {
	h := NewHandler()
	h.HitMap.Add("btn", Rect{X: 0, Y: 0, W: 10, H: 10}, nil)

	h.HandleMouse(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      5,
		Y:      5,
	})

	action := h.HandleMouse(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      5,
		Y:      5,
	})
	if action.Type != ActionDoubleClick {
		t.Errorf("expected ActionDoubleClick, got %d", action.Type)
	}
}

func TestHandleMouse_ScrollUp(t *testing.T) {
	h := NewHandler()
	h.HitMap.Add("content", Rect{X: 0, Y: 0, W: 80, H: 24}, nil)

	action := h.HandleMouse(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelUp,
		X:      5,
		Y:      5,
	})
	if action.Type != ActionScrollUp {
		t.Errorf("expected ActionScrollUp, got %d", action.Type)
	}
	if action.Delta != -3 {
		t.Errorf("expected delta -3, got %d", action.Delta)
	}
}

func TestHandleMouse_ScrollDown(t *testing.T) {
	h := NewHandler()

	action := h.HandleMouse(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelDown,
		X:      5,
		Y:      5,
	})
	if action.Type != ActionScrollDown {
		t.Errorf("expected ActionScrollDown, got %d", action.Type)
	}
	if action.Delta != 3 {
		t.Errorf("expected delta 3, got %d", action.Delta)
	}
}

func TestHandleMouse_ShiftScrollHorizontal(t *testing.T) {
	h := NewHandler()

	action := h.HandleMouse(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelUp,
		Shift:  true,
		X:      5,
		Y:      5,
	})
	if action.Type != ActionScrollLeft {
		t.Errorf("expected ActionScrollLeft for shift+wheel up, got %d", action.Type)
	}

	action = h.HandleMouse(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelDown,
		Shift:  true,
		X:      5,
		Y:      5,
	})
	if action.Type != ActionScrollRight {
		t.Errorf("expected ActionScrollRight for shift+wheel down, got %d", action.Type)
	}
}

func TestHandleMouse_NativeHorizontalScroll(t *testing.T) {
	h := NewHandler()

	action := h.HandleMouse(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelLeft,
		X:      5,
		Y:      5,
	})
	if action.Type != ActionScrollRight {
		t.Errorf("expected ActionScrollRight for WheelLeft (Mac natural), got %d", action.Type)
	}

	action = h.HandleMouse(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelRight,
		X:      5,
		Y:      5,
	})
	if action.Type != ActionScrollLeft {
		t.Errorf("expected ActionScrollLeft for WheelRight (Mac natural), got %d", action.Type)
	}
}

func TestHandleMouse_DragMotion(t *testing.T) {
	h := NewHandler()
	h.StartDrag(10, 10, "divider", 100)

	action := h.HandleMouse(tea.MouseMsg{
		Action: tea.MouseActionMotion,
		X:      20,
		Y:      15,
	})
	if action.Type != ActionDrag {
		t.Errorf("expected ActionDrag, got %d", action.Type)
	}
	if action.DragDX != 10 || action.DragDY != 5 {
		t.Errorf("expected drag delta (10, 5), got (%d, %d)", action.DragDX, action.DragDY)
	}
}

func TestHandleMouse_DragRelease(t *testing.T) {
	h := NewHandler()
	h.StartDrag(10, 10, "divider", 100)

	action := h.HandleMouse(tea.MouseMsg{
		Action: tea.MouseActionRelease,
	})
	if action.Type != ActionDragEnd {
		t.Errorf("expected ActionDragEnd, got %d", action.Type)
	}
	if h.IsDragging() {
		t.Error("should not be dragging after release")
	}
}

func TestHandleMouse_Hover(t *testing.T) {
	h := NewHandler()
	h.HitMap.Add("btn", Rect{X: 0, Y: 0, W: 10, H: 10}, nil)

	action := h.HandleMouse(tea.MouseMsg{
		Action: tea.MouseActionMotion,
		X:      5,
		Y:      5,
	})
	if action.Type != ActionHover {
		t.Errorf("expected ActionHover, got %d", action.Type)
	}
	if action.Region == nil || action.Region.ID != "btn" {
		t.Error("expected hover over region 'btn'")
	}

	action = h.HandleMouse(tea.MouseMsg{
		Action: tea.MouseActionMotion,
		X:      50,
		Y:      50,
	})
	if action.Type != ActionHover {
		t.Errorf("expected ActionHover even for miss, got %d", action.Type)
	}
	if action.Region != nil {
		t.Error("expected nil region for hover miss")
	}
}
