package trinkets

// addHostWindowMenuItems writes Minimize and Zoom into a menu it does not
// own, so it has to look at what is already there. createSystemMenu closes
// its last group with a separator before Exit Desktop, and inserting another
// unconditionally drew two rules in a row.
//
// A doubled rule is the kind of thing that survives for a long time: it only
// appears when the themed frame is active, it breaks nothing, and it reads as
// an empty group rather than as a bug.

import (
	"testing"

	"github.com/phroun/kittytk/core"
)

// themedFrameDesktop is a desktop with the themed frame active, which is what
// gates addHostWindowMenuItems.
func themedFrameDesktop() *Desktop {
	d := NewDesktop()
	d.graphicalFrames = true
	d.surface = &fakeNativeSurface{size: core.UnitSize{Width: 800, Height: 600}, pxW: 800, pxH: 600}
	return d
}

func TestHostWindowMenuItemsDoNotDoubleTheSeparator(t *testing.T) {
	d := themedFrameDesktop()
	if !d.themedFrameActive() {
		t.Fatal("themed frame is not active; the insert would not run")
	}

	d.addHostWindowMenuItems()

	items := d.systemMenu.Items()
	for i := 1; i < len(items); i++ {
		if items[i-1].Separator && items[i].Separator {
			var names []string
			for _, it := range items {
				if it.Separator {
					names = append(names, "---")
					continue
				}
				names = append(names, it.Text)
			}
			t.Fatalf("two separators in a row at %d..%d: %v", i-1, i, names)
		}
	}

	// The items still landed, still above Exit, and still with a rule
	// separating them from what precedes.
	var minAt, zoomAt, exitAt = -1, -1, -1
	for i, it := range items {
		switch it.Text {
		case "Minimize":
			minAt = i
		case "Zoom":
			zoomAt = i
		case "Exit Desktop":
			exitAt = i
		}
	}
	if minAt < 0 || zoomAt != minAt+1 || exitAt != zoomAt+1 {
		t.Fatalf("Minimize/Zoom/Exit at %d/%d/%d, want consecutive", minAt, zoomAt, exitAt)
	}
	if minAt == 0 || !items[minAt-1].Separator {
		t.Error("Minimize has no separator above it")
	}
}

// Calling it twice is not something the desktop does, but the guard reads the
// item above the insert point, and that item is a different one the second
// time. Left unpinned, a later change to the group order could turn the
// no-double guard into a no-separator-at-all.
func TestHostWindowMenuItemsSeparatorSurvivesAReinsert(t *testing.T) {
	d := themedFrameDesktop()
	d.addHostWindowMenuItems()
	d.addHostWindowMenuItems()

	items := d.systemMenu.Items()
	for i := 1; i < len(items); i++ {
		if items[i-1].Separator && items[i].Separator {
			t.Fatalf("two separators in a row at %d..%d after a reinsert", i-1, i)
		}
	}
}
