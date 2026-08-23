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

// accessoriesItem finds the Desktop Accessories item and its submenu.
func accessoriesItem(t *testing.T, d *Desktop) *MenuItem {
	t.Helper()
	for _, it := range d.systemMenu.Items() {
		if strings.Contains(it.Text, "Desktop Accessories") {
			return it
		}
	}
	t.Fatal("Desktop Accessories item not found in the system menu")
	return nil
}

func TestDesktopAccessoriesIsASubMenu(t *testing.T) {
	d := NewDesktop()
	it := accessoriesItem(t, d)

	// It used to be a disabled placeholder; it is now a real submenu, which
	// means it must be enabled or the submenu can never be opened.
	if !it.Enabled {
		t.Error("Desktop Accessories is disabled, so its submenu is unreachable")
	}
	if it.SubMenu == nil {
		t.Fatal("Desktop Accessories has no submenu")
	}
	var viewer *MenuItem
	for _, sub := range it.SubMenu.Items() {
		if strings.Contains(sub.Text, "Event Viewer") {
			viewer = sub
		}
	}
	if viewer == nil {
		t.Fatal("Event Viewer not found under Desktop Accessories")
	}
	if viewer.OnTriggered == nil {
		t.Error("Event Viewer item is not wired")
	}
}

func TestEventViewerOpensOnceAndLogs(t *testing.T) {
	d := NewDesktop()
	d.windowManager = window.NewWindowManager()

	var viewer *MenuItem
	for _, sub := range accessoriesItem(t, d).SubMenu.Items() {
		if strings.Contains(sub.Text, "Event Viewer") {
			viewer = sub
		}
	}

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

func TestEventViewerStopsLoggingWhenClosed(t *testing.T) {
	d := NewDesktop()
	d.windowManager = window.NewWindowManager()

	var viewer *MenuItem
	for _, sub := range accessoriesItem(t, d).SubMenu.Items() {
		if strings.Contains(sub.Text, "Event Viewer") {
			viewer = sub
		}
	}
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
