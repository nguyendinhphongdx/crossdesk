//go:build darwin

// Native macOS shell: menu-bar status item and a WKWebView window.
#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>
#include "shell_darwin.h"


extern void goQuit(void);

@interface XDShell : NSObject <NSApplicationDelegate, NSWindowDelegate, WKUIDelegate, WKNavigationDelegate>
@property (strong) NSWindow *window;
@property (strong) WKWebView *web;
@property (strong) NSStatusItem *item;
@property (copy) NSString *url;
@end

@implementation XDShell
- (void)showWindow:(id)sender {
	// A Dock icon only while the window is open; otherwise just the menu-bar icon.
	[NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
	if (!self.window) {
		NSRect r = NSMakeRect(0, 0, 1120, 800);
		self.window = [[NSWindow alloc] initWithContentRect:r
			styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable | NSWindowStyleMaskMiniaturizable | NSWindowStyleMaskResizable
			backing:NSBackingStoreBuffered defer:NO];
		self.window.title = @"Crossdesk";
		self.window.releasedWhenClosed = NO;
		self.window.delegate = self;
		self.window.minSize = NSMakeSize(720, 520);
		WKWebViewConfiguration *cfg = [WKWebViewConfiguration new];
		self.web = [[WKWebView alloc] initWithFrame:r configuration:cfg];
		self.web.UIDelegate = self;
		self.web.navigationDelegate = self;
		[self.web setValue:@NO forKey:@"drawsBackground"];
		self.window.contentView = self.web;
		[self.web loadRequest:[NSURLRequest requestWithURL:[NSURL URLWithString:self.url]]];
		[self.window center];
		[self.window setFrameAutosaveName:@"CrossdeskMainWindow"];
	}
	[self.window makeKeyAndOrderFront:nil];
	[NSApp activateIgnoringOtherApps:YES];
}

- (void)windowWillClose:(NSNotification *)n {
	[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
}

- (BOOL)applicationShouldHandleReopen:(NSApplication *)app hasVisibleWindows:(BOOL)visible {
	[self showWindow:nil];
	return YES;
}

- (void)quit:(id)sender { goQuit(); }

- (NSApplicationTerminateReply)applicationShouldTerminate:(NSApplication *)sender {
	goQuit(); // the Go side exits the process after cleaning up
	return NSTerminateCancel;
}

// window.alert / window.confirm from the control page.
- (void)webView:(WKWebView *)w runJavaScriptAlertPanelWithMessage:(NSString *)msg
	initiatedByFrame:(WKFrameInfo *)frame completionHandler:(void (^)(void))done {
	NSAlert *a = [NSAlert new];
	a.messageText = msg;
	[a addButtonWithTitle:@"OK"];
	[a beginSheetModalForWindow:self.window completionHandler:^(NSModalResponse r) { done(); }];
}

- (void)webView:(WKWebView *)w runJavaScriptConfirmPanelWithMessage:(NSString *)msg
	initiatedByFrame:(WKFrameInfo *)frame completionHandler:(void (^)(BOOL))done {
	NSAlert *a = [NSAlert new];
	a.messageText = msg;
	[a addButtonWithTitle:@"OK"];
	[a addButtonWithTitle:@"Huỷ"];
	[a beginSheetModalForWindow:self.window completionHandler:^(NSModalResponse r) {
		done(r == NSAlertFirstButtonReturn);
	}];
}

// Links that leave the control page open in the default browser.
- (void)webView:(WKWebView *)w decidePolicyForNavigationAction:(WKNavigationAction *)action
	decisionHandler:(void (^)(WKNavigationActionPolicy))decide {
	NSURL *u = action.request.URL;
	NSString *host = u.host ?: @"";
	if (action.navigationType == WKNavigationTypeLinkActivated &&
	    !([host isEqualToString:@"127.0.0.1"] || [host isEqualToString:@"localhost"])) {
		[[NSWorkspace sharedWorkspace] openURL:u];
		decide(WKNavigationActionPolicyCancel);
		return;
	}
	decide(WKNavigationActionPolicyAllow);
}

- (WKWebView *)webView:(WKWebView *)w createWebViewWithConfiguration:(WKWebViewConfiguration *)c
	forNavigationAction:(WKNavigationAction *)action windowFeatures:(WKWindowFeatures *)f {
	if (action.request.URL) [[NSWorkspace sharedWorkspace] openURL:action.request.URL];
	return nil;
}
@end

static XDShell *gShell;

static NSMenuItem *addItem(NSMenu *m, NSString *title, SEL action, NSString *key, id target) {
	NSMenuItem *it = [m addItemWithTitle:title action:action keyEquivalent:key];
	if (target) it.target = target;
	return it;
}

// Main menu (shown while the window is open) so Cmd+C/V/A/W/Q work.
static void buildMainMenu(void) {
	NSMenu *bar = [NSMenu new];
	NSMenu *app = [NSMenu new];
	addItem(app, @"Mở bảng điều khiển", @selector(showWindow:), @"", gShell);
	[app addItem:[NSMenuItem separatorItem]];
	addItem(app, @"Ẩn Crossdesk", @selector(hide:), @"h", nil);
	[app addItem:[NSMenuItem separatorItem]];
	addItem(app, @"Thoát Crossdesk", @selector(quit:), @"q", gShell);
	[bar addItemWithTitle:@"" action:nil keyEquivalent:@""].submenu = app;

	NSMenu *edit = [[NSMenu alloc] initWithTitle:@"Sửa"];
	addItem(edit, @"Hoàn tác", @selector(undo:), @"z", nil);
	addItem(edit, @"Làm lại", @selector(redo:), @"Z", nil);
	[edit addItem:[NSMenuItem separatorItem]];
	addItem(edit, @"Cắt", @selector(cut:), @"x", nil);
	addItem(edit, @"Sao chép", @selector(copy:), @"c", nil);
	addItem(edit, @"Dán", @selector(paste:), @"v", nil);
	addItem(edit, @"Chọn tất cả", @selector(selectAll:), @"a", nil);
	[bar addItemWithTitle:@"Sửa" action:nil keyEquivalent:@""].submenu = edit;

	NSMenu *win = [[NSMenu alloc] initWithTitle:@"Cửa sổ"];
	addItem(win, @"Thu nhỏ", @selector(performMiniaturize:), @"m", nil);
	addItem(win, @"Đóng", @selector(performClose:), @"w", nil);
	[bar addItemWithTitle:@"Cửa sổ" action:nil keyEquivalent:@""].submenu = win;
	NSApp.mainMenu = bar;
	NSApp.windowsMenu = win;
}

void runShell(const char *url, const void *icon, int iconLen, int show) {
	@autoreleasepool {
		[NSApplication sharedApplication];
		[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
		gShell = [XDShell new];
		gShell.url = [NSString stringWithUTF8String:url];
		NSApp.delegate = gShell;
		buildMainMenu();

		NSStatusItem *it = [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];
		NSImage *img = [[NSImage alloc] initWithData:[NSData dataWithBytes:icon length:iconLen]];
		img.size = NSMakeSize(18, 18);
		img.template = YES; // follows light/dark menu bar automatically
		it.button.image = img;
		it.button.toolTip = @"Crossdesk";
		NSMenu *m = [NSMenu new];
		addItem(m, @"Mở Crossdesk", @selector(showWindow:), @"", gShell);
		[m addItem:[NSMenuItem separatorItem]];
		addItem(m, @"Thoát Crossdesk", @selector(quit:), @"q", gShell);
		it.menu = m;
		gShell.item = it;

		if (show) [gShell showWindow:nil];
		[NSApp run];
	}
}

void showWindowAsync(void) {
	dispatch_async(dispatch_get_main_queue(), ^{ [gShell showWindow:nil]; });
}
