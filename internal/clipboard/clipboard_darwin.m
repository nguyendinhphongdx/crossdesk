//go:build darwin

#import <Cocoa/Cocoa.h>
#include <stdlib.h>
#include <string.h>
#include "clipboard_darwin.h"

static void *copyBytes(const void *src, size_t n) {
	void *p = malloc(n ? n : 1);
	if (p && n) memcpy(p, src, n);
	return p;
}

// Returns 0 = nothing usable, 1 = UTF-8 text, 2 = PNG. *out must be freed.
int cdClipboardRead(void **out, long *len) {
	@autoreleasepool {
		NSPasteboard *pb = [NSPasteboard generalPasteboard];
		NSString *s = [pb stringForType:NSPasteboardTypeString];
		if (s.length > 0) {
			const char *u = [s UTF8String];
			size_t n = strlen(u);
			*out = copyBytes(u, n);
			*len = (long)n;
			return 1;
		}
		NSData *png = [pb dataForType:NSPasteboardTypePNG];
		if (!png) {
			NSData *tiff = [pb dataForType:NSPasteboardTypeTIFF];
			if (tiff) {
				NSBitmapImageRep *rep = [NSBitmapImageRep imageRepWithData:tiff];
				png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
			}
		}
		if (png.length > 0) {
			*out = copyBytes(png.bytes, png.length);
			*len = (long)png.length;
			return 2;
		}
		return 0;
	}
}

int cdClipboardWrite(int kind, const void *data, long len) {
	@autoreleasepool {
		NSPasteboard *pb = [NSPasteboard generalPasteboard];
		[pb clearContents];
		if (kind == 1) {
			NSString *s = [[NSString alloc] initWithBytes:data length:len encoding:NSUTF8StringEncoding];
			return s && [pb setString:s forType:NSPasteboardTypeString] ? 0 : -1;
		}
		NSData *png = [NSData dataWithBytes:data length:len];
		BOOL ok = [pb setData:png forType:NSPasteboardTypePNG];
		// Many Mac apps still look for TIFF.
		NSBitmapImageRep *rep = [NSBitmapImageRep imageRepWithData:png];
		if (rep) [pb setData:[rep TIFFRepresentation] forType:NSPasteboardTypeTIFF];
		return ok ? 0 : -1;
	}
}
