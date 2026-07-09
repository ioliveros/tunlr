#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>

extern void tunlrStartTray(void);
extern void tunlrReopen(void);

void tunlrDispatchStartTray(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		tunlrStartTray();
	});
}

static BOOL tunlrHandleReopen(id self, SEL _cmd, NSApplication *app, BOOL hasVisibleWindows) {
	tunlrReopen();
	return YES;
}

// systray replaces Wails' app delegate (which handled Dock reopen), so add the
// reopen method to whatever delegate is now installed to restore a hidden window.
void tunlrInstallReopenHandler(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		id delegate = [[NSApplication sharedApplication] delegate];
		if (delegate != nil) {
			class_addMethod([delegate class],
				@selector(applicationShouldHandleReopen:hasVisibleWindows:),
				(IMP)tunlrHandleReopen, "c@:@c");
		}
	});
}
