# share-device

Dùng **một bàn phím + trackpad/chuột của MacBook** để điều khiển máy **Windows** trong cùng mạng LAN.
Đẩy con trỏ ra mép màn hình Mac → con trỏ "nhảy" sang Windows; kéo ngược lại → quay về Mac.

```
┌──────────── MacBook (server) ────────────┐        TCP 24800, mã hoá AES-GCM        ┌──── Windows (client) ────┐
│ CGEventTap bắt chuột/phím                 │ ─────────────────────────────────────► │ SendInput giả lập        │
│ chạm mép phải → chặn input local, gửi đi  │                                         │ chuột + phím             │
└───────────────────────────────────────────┘                                         └──────────────────────────┘
```

## Build

Cần Go ≥ 1.24. Build trên Mac ra luôn cả file `.exe` cho Windows:

```bash
make            # hoặc chạy tay:
go build -o dist/share-device ./cmd/share-device
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o dist/share-device.exe ./cmd/share-device
```

## Chạy

**1. Trên Mac** (máy có bàn phím/trackpad):

```bash
./dist/share-device server -password "mat-khau-bi-mat" -side right
```

- `-side`: Windows nằm ở phía nào của màn hình Mac: `left`, `right`, `top`, `bottom`.
- `-speed`: tốc độ con trỏ bên Windows (mặc định 1.5).
- Lần đầu macOS sẽ hỏi quyền: vào **System Settings → Privacy & Security → Accessibility**
  (và **Input Monitoring**) bật cho Terminal/iTerm (app đang chạy lệnh), rồi chạy lại.
- Server in ra IP của Mac để bạn dùng ở bước 2.

**2. Trên Windows**: copy `dist/share-device.exe` sang và chạy:

```powershell
.\share-device.exe client -server 192.168.1.10 -password "mat-khau-bi-mat"
```

- Lần đầu Windows Firewall có thể hỏi — cho phép (client chỉ kết nối ra, nhưng bên Mac cần cho phép kết nối vào cổng 24800).
- `-swap-cmd-ctrl` (mặc định bật trên Windows): **Cmd → Ctrl** để Cmd+C/Cmd+V copy/paste như trên Mac; Ctrl → phím Windows.
  Tắt bằng `-swap-cmd-ctrl=false`.

Có thể dùng biến môi trường `SHARE_DEVICE_PASSWORD` thay cho `-password`.

## Tính năng

- Chuyển máy bằng cách đẩy con trỏ qua mép màn hình (không chuyển khi đang giữ chuột/kéo thả).
- Chuột: di chuyển, click trái/phải/giữa, nút back/forward, cuộn dọc + ngang (trackpad cuộn mượt).
- Bàn phím: map theo vị trí phím (USB HID), nên layout bàn phím bên Windows quyết định ký tự.
- Phím đang giữ được tự nhả khi chuyển máy hoặc mất kết nối → không bị kẹt phím.
- Mã hoá toàn bộ (PBKDF2 + AES-256-GCM, salt mới mỗi phiên): mật khẩu bạn gõ sang Windows không đi dạng plaintext trong LAN.
- Client tự kết nối lại khi mất mạng; mất kết nối thì Mac lập tức lấy lại quyền điều khiển.

## Giới hạn hiện tại

- Chỉ Mac làm server (bắt input), Windows/Mac làm client. Chưa hỗ trợ dùng bàn phím Windows điều khiển Mac.
- Chưa chia sẻ clipboard, chưa kéo-thả file.
- Một client tại một thời điểm.
- Màn hình khoá / UAC (secure desktop) của Windows không nhận input giả lập — giới hạn của Windows.
- Phím media trên hàng F của Mac (độ sáng, play/pause) chưa được chuyển.
- Khi đang điều khiển Windows, con trỏ Mac vẫn hiện (đứng yên) ở mép màn hình.

## Cấu trúc

| Thư mục | Vai trò |
|---|---|
| `cmd/share-device` | CLI `server` / `client` |
| `internal/capture` | Bắt input toàn cục trên macOS (CGEventTap, cgo) |
| `internal/server` | Logic chuyển máy theo mép màn hình, theo dõi toạ độ ảo |
| `internal/client` | Nhận message, giả lập input, tự kết nối lại |
| `internal/inject` | Giả lập input: Windows (`SendInput`, không cần cgo), macOS (CGEvent) |
| `internal/proto` | Message nhị phân + kênh mã hoá |
| `internal/keymap` | Bảng mã phím macOS ↔ USB HID ↔ Windows scancode |

Chạy test: `go test ./...`
