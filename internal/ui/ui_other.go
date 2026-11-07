//go:build !darwin && !windows

package ui

// Run opens the control page in the browser and blocks forever. A native
// window on Linux would require GTK/WebKitGTK on every machine, so the
// browser is used instead.
func Run(o Options) {
	lastURL = o.URL
	if o.ShowWindow {
		OpenBrowser(o.URL)
	}
	select {}
}

var lastURL string

// Show opens the control page in the browser.
func Show() {
	if lastURL != "" {
		OpenBrowser(lastURL)
	}
}
