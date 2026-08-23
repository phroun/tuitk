package trinkets

// The Event Viewer accessory, reached through Desktop Accessories in the
// system (Ψ) menu.
//
// Two things here are easy to get wrong and invisible if they are. Opening it
// twice must raise the one window rather than stacking a second; and because
// AddEventFilter has no counterpart, the filter it installs is permanent, so
// closing the window has to stop the logging by some other means than removing
// it.

import (
	"strings"
	"testing"

	"github.com/phroun/kittytk/core"
	"github.com/phroun/kittytk/objects/window"
)

// eventViewerItem finds the Event Viewer item in the system menu.
func eventViewerItem(t *testing.T, d *Desktop) *MenuItem {
	t.Helper()
	for _, it := range d.systemMenu.Items() {
		if strings.Contains(it.Text, "Event Viewer") {
			return it
		}
	}
	t.Fatal("Event Viewer not found in the system menu")
	return nil
}

// The accessories are a flat run under a disabled heading rather than a
// submenu. The heading must stay disabled - an enabled one would look like a
// command that does nothing - and the accessories under it must NOT be, which
// is the pairing this checks.
func TestDesktopAccessoriesHeadingAndItems(t *testing.T) {
	d := NewDesktop()

	items := d.systemMenu.Items()
	heading := -1
	for i, it := range items {
		if strings.Contains(it.Text, "Desktop Accessories") {
			heading = i
		}
	}
	if heading < 0 {
		t.Fatal("Desktop Accessories heading not found in the system menu")
	}
	if items[heading].Enabled {
		t.Error("the Desktop Accessories heading is enabled; it would read as a command")
	}
	if items[heading].SubMenu != nil {
		t.Error("the heading carries a submenu; the accessories are meant to be flat")
	}

	// The first accessory sits directly below the heading.
	if heading+1 >= len(items) {
		t.Fatal("nothing follows the Desktop Accessories heading")
	}
	first := items[heading+1]
	if !strings.Contains(first.Text, "Event Viewer") {
		t.Errorf("item below the heading is %q, want Event Viewer", first.Text)
	}
	if !first.Enabled {
		t.Error("Event Viewer is disabled")
	}
	if first.OnTriggered == nil {
		t.Error("Event Viewer item is not wired")
	}
}

func TestEventViewerOpensOnceAndLogs(t *testing.T) {
	d := NewDesktop()
	d.windowManager = window.NewWindowManager()

	viewer := eventViewerItem(t, d)

	viewer.OnTriggered()
	wins := d.windowManager.Windows()
	if len(wins) != 1 {
		t.Fatalf("opened %d windows, want 1", len(wins))
	}
	if got := wins[0].Title(); got != "Event Viewer" {
		t.Errorf("title = %q, want Event Viewer", got)
	}

	// Triggering again raises the same window rather than opening a second.
	viewer.OnTriggered()
	if got := len(d.windowManager.Windows()); got != 1 {
		t.Fatalf("second trigger left %d windows, want 1", got)
	}

	v := d.eventViewer
	if v == nil {
		t.Fatal("desktop is not tracking the open viewer")
	}

	// An event reaches the log through the desktop's filter, and is not
	// consumed on the way: the filter is an observer.
	if d.filterEvent(core.KeyPressEvent{Key: "a", Text: "a"}) {
		t.Error("the viewer's filter consumed the event")
	}
	if got := len(v.tree.RootItems()); got != 1 {
		t.Fatalf("logged %d rows, want 1", got)
	}

	// Mouse events are filtered out by default, so the noisy ones do not
	// bury the keystroke being looked for.
	d.filterEvent(core.MouseMoveEvent{X: 1, Y: 1})
	if got := len(v.tree.RootItems()); got != 1 {
		t.Errorf("mouse move logged with the filter off: %d rows", got)
	}
	v.showMouse = true
	d.filterEvent(core.MouseMoveEvent{X: 2, Y: 2})
	if got := len(v.tree.RootItems()); got != 2 {
		t.Errorf("mouse move not logged with the filter on: %d rows", got)
	}
}

// The viewer is opened OVER the program being watched, so a window that
// covers the desktop defeats the point. Its preferred size is what the tree's
// columns want, which is wider than most desktops - the cap is what decides
// the size in practice, and it is the part worth pinning.
func TestEventViewerFitsInsideTheDesktop(t *testing.T) {
	for _, size := range []core.UnitSize{
		{Width: 400, Height: 300},   // smaller than the preferred size
		{Width: 4000, Height: 3000}, // larger than it
	} {
		d := NewDesktop()
		d.SetBounds(core.UnitRect{Width: size.Width, Height: size.Height})
		d.windowManager = window.NewWindowManager()
		d.windowManager.SetDesktop(d)

		area := d.windowManager.ClientArea()
		if area.Width <= 0 || area.Height <= 0 {
			t.Fatalf("client area is %dx%d; the cap would not be exercised",
				area.Width, area.Height)
		}

		eventViewerItem(t, d).OnTriggered()

		b := d.windowManager.Windows()[0].Bounds()
		if b.Width > area.Width || b.Height > area.Height {
			t.Errorf("in a %dx%d client area the viewer is %dx%d",
				area.Width, area.Height, b.Width, b.Height)
		}
		// And it stays on screen rather than being centred off the edge.
		if b.X < area.X || b.Y < area.Y ||
			b.X+b.Width > area.X+area.Width || b.Y+b.Height > area.Y+area.Height {
			t.Errorf("in a %dx%d client area the viewer sits at %d,%d %dx%d",
				area.Width, area.Height, b.X, b.Y, b.Width, b.Height)
		}
	}
}

func TestEventViewerStopsLoggingWhenClosed(t *testing.T) {
	d := NewDesktop()
	d.windowManager = window.NewWindowManager()

	viewer := eventViewerItem(t, d)
	viewer.OnTriggered()
	first := d.eventViewer
	win := first.win

	win.Close()
	if d.eventViewer != nil {
		t.Fatal("closing the window did not clear the desktop's reference")
	}

	// The filter is still installed - there is no way to remove one - so the
	// thing being tested is that it now does nothing rather than logging into
	// a window nobody can see.
	before := len(first.tree.RootItems())
	d.filterEvent(core.KeyPressEvent{Key: "b", Text: "b"})
	if got := len(first.tree.RootItems()); got != before {
		t.Errorf("closed viewer still logging: %d rows, was %d", got, before)
	}

	// Opening it again builds a fresh viewer and logs into that one, without
	// installing a second filter.
	viewer.OnTriggered()
	second := d.eventViewer
	if second == nil || second == first {
		t.Fatal("reopening did not build a new viewer")
	}
	d.filterEvent(core.KeyPressEvent{Key: "c", Text: "c"})
	if got := len(second.tree.RootItems()); got != 1 {
		t.Errorf("reopened viewer logged %d rows, want 1", got)
	}
	if got := len(first.tree.RootItems()); got != before {
		t.Errorf("the closed viewer logged again: %d rows", got)
	}
}
