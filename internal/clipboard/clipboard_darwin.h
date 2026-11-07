#ifndef CROSSDESK_CLIPBOARD_DARWIN_H
#define CROSSDESK_CLIPBOARD_DARWIN_H
int cdClipboardRead(void **out, long *len);
int cdClipboardWrite(int kind, const void *data, long len);
#endif
