//go:build linux && cgo && gtk3

package desktop

/*
#cgo pkg-config: gtk+-3.0
#include <gtk/gtk.h>
#include <cairo.h>

typedef struct {
	int x;
	int y;
	int width;
	int height;
} NexusHUDInputRect;

static void nexusHUDApplyInputRegions(GtkWidget *widget, const NexusHUDInputRect *rects, size_t count) {
	if (widget == NULL) {
		return;
	}
	if (count == 0) {
		gtk_widget_input_shape_combine_region(widget, NULL);
		return;
	}
	cairo_region_t *region = cairo_region_create();
	for (size_t i = 0; i < count; i++) {
		if (rects[i].width <= 0 || rects[i].height <= 0) {
			continue;
		}
		cairo_rectangle_int_t rectangle = {
			.x = rects[i].x,
			.y = rects[i].y,
			.width = rects[i].width,
			.height = rects[i].height,
		};
		cairo_region_union_rectangle(region, &rectangle);
	}
	gtk_widget_input_shape_combine_region(widget, region);
	cairo_region_destroy(region);
}
*/
import "C"

import "github.com/wailsapp/wails/v3/pkg/application"

// The Linux GTK backend has no per-pixel SetIgnoreMouseEvents support. GTK's
// input shape provides the equivalent behavior by forwarding pointer events
// outside the renderer-reported interactive rectangles to the window below.
func hudHitTestingSupported() bool { return false }

func hudPointerWithinRegions(_ application.Window, _ []HUDHitRegion) bool { return true }

func applyHUDHitRegions(window application.Window, regions []HUDHitRegion) {
	native := window.NativeWindow()
	if native == nil {
		return
	}
	inputRects := hudRegionsToInputRectangles(regions)
	cRects := make([]C.NexusHUDInputRect, len(inputRects))
	for i, rect := range inputRects {
		cRects[i] = C.NexusHUDInputRect{
			x: C.int(rect.X), y: C.int(rect.Y),
			width: C.int(rect.Width), height: C.int(rect.Height),
		}
	}

	application.InvokeSync(func() {
		var rectPointer *C.NexusHUDInputRect
		if len(cRects) > 0 {
			rectPointer = &cRects[0]
		}
		C.nexusHUDApplyInputRegions(
			(*C.GtkWidget)(native), rectPointer, C.size_t(len(cRects)),
		)
	})
}

func resetHUDHitRegions(window application.Window) {
	applyHUDHitRegions(window, nil)
}
