<p align="center"><img src="assets/brand/crossdesk-mark-256.png" width="96" alt="Crossdesk"></p>

<h1 align="center">Crossdesk</h1>

Dùng **một bàn phím và một con chuột/trackpad** cho nhiều máy tính trong cùng mạng LAN — **macOS, Windows, Linux**.
Đẩy con trỏ qua mép màn hình là sang máy bên cạnh. Hai chiều: máy nào làm máy chủ cũng được.

- Tự tìm các máy trong mạng (cùng mật khẩu nhóm = cùng nhóm)
- Chọn máy chủ và **kéo thả sắp xếp vị trí** màn hình trong bảng điều khiển
- Tự đổi Cmd ⇄ Ctrl giữa Mac và Windows/Linux
- Mã hoá toàn bộ (PBKDF2 + AES-256-GCM), gói quảng bá ký HMAC

## Cài đặt

Tải ở trang web của dự án (tự nhận diện hệ điều hành) hoặc mục **Releases** trên GitHub.

| Hệ điều hành | File | Ghi chú |
|---|---|---|
| macOS 11+ (Apple Silicon & Intel) | `Crossdesk-macOS-universal.zip` | Kéo vào Applications. Lần đầu: chuột phải → Open. Cấp quyền **Accessibility** (+ Input Monitoring). |
| Windows 10/11 x64 / ARM64 | `Crossdesk-Windows-x64.exe` / `-arm64.exe` | Chạy trực tiếp. Cho phép qua Firewall (mạng Private). |
| Linux x64 / ARM64 (X11 & Wayland) | `Crossdesk-Linux-x64.tar.gz` / `-arm64` | `sudo ./install.sh`, rồi đăng xuất/đăng nhập lại (cấp quyền `/dev/uinput` + nhóm `input`). |

## Sử dụng

1. Mở Crossdesk trên mọi máy — bảng điều khiển tự mở trong trình duyệt (`http://127.0.0.1:24801`).
2. Đặt **cùng một mật khẩu nhóm** trên các máy.
3. Bấm **Làm máy chủ** ở máy có bàn phím/chuột bạn muốn dùng. Các máy khác tự thành máy khách.
4. Kéo thả các màn hình trong lưới cho giống vị trí thật. Đẩy chuột qua mép để chuyển máy.

Cổng mạng: TCP **24800** (input, mã hoá), UDP **24802** (tìm máy), `127.0.0.1:24801` (bảng điều khiển, chỉ máy cục bộ).

## Kiến trúc

```
                ┌─────────────── mỗi máy chạy cùng một ứng dụng ───────────────┐
 trình duyệt ◄──┤ web UI (127.0.0.1, token)                                    │
                │ agent: tìm máy (UDP, HMAC) · cấu hình nhóm (đồng bộ LWW)      │
                │   ├─ máy chủ: capture → server (lưới màn hình) ──TCP AES-GCM──┼──► máy khách: client → inject
                │   └─ máy khách: client → inject                                │
                └──────────────────────────────────────────────────────────────┘
```

| Thư mục | Vai trò |
|---|---|
| `cmd/crossdesk` | Ứng dụng: khởi động agent + web UI, mở trình duyệt, một phiên bản duy nhất |
| `internal/agent` | Tìm máy, đồng bộ cấu hình nhóm (máy chủ, sơ đồ), tự chuyển vai trò |
| `internal/server` | Định tuyến con trỏ qua lưới màn hình, nhiều máy khách |
| `internal/client` | Nhận input, giả lập, tự kết nối lại |
| `internal/capture` | Bắt input: macOS (CGEventTap), Windows (low-level hooks), Linux (evdev + EVIOCGRAB) |
| `internal/inject` | Giả lập input: macOS (CGEvent), Windows (SendInput), Linux (uinput) |
| `internal/linuxinput` | uinput/evdev thuần Go, không cgo |
| `internal/proto` | Giao thức nhị phân + kênh mã hoá |
| `internal/keymap` | Bảng mã phím macOS ↔ USB HID ↔ Windows scancode ↔ Linux evdev |
| `internal/web` | Bảng điều khiển (HTML nhúng) |
| `site/` | Trang giới thiệu + trang tải về (Vite, React, Tailwind, shadcn/ui) |
| `packaging/` | Info.plist macOS, script cài đặt Linux, udev rule |

## Build

```bash
make mac                   # dist/Crossdesk-macOS-universal.zip (cần macOS)
make windows-linux         # dist/Crossdesk-Windows-*.exe, dist/Crossdesk-Linux-*.tar.gz
make test
make site                  # site/dist
```

Icon và manifest của file `.exe` Windows nằm trong `cmd/crossdesk/rsrc_windows_*.syso`. Tạo lại khi đổi logo:

```bash
go run github.com/tc-hib/go-winres@v0.3.3 simply --icon packaging/windows/crossdesk-256.png --manifest gui \
  --product-name Crossdesk --file-description Crossdesk --original-filename Crossdesk.exe \
  --arch amd64,arm64 --out cmd/crossdesk/rsrc
```

## Phát hành (CI/CD)

| Workflow | Khi nào | Làm gì |
|---|---|---|
| `ci.yml` | push / PR | Test Go trên Linux, macOS, Windows · test uinput thật · cross-compile · build web |
| `release.yml` | push tag `v*` | Test → build macOS (universal, ký ad-hoc), Windows, Linux → `SHA256SUMS.txt` → tạo GitHub Release |
| `pages.yml` | push `site/**` lên main | Build web với đúng base path + tên repo → deploy GitHub Pages |

Phát hành bản mới:

```bash
git tag v0.3.0
git push origin v0.3.0
```

Thiết lập một lần trên GitHub: **Settings → Pages → Source: GitHub Actions**.
Trang tải về luôn trỏ tới `releases/latest/download/<file>`, nên không cần build lại web sau mỗi lần phát hành.

## Giới hạn hiện tại

- Chưa chia sẻ clipboard / kéo thả file.
- App macOS chỉ ký ad-hoc (chưa notarize) → Gatekeeper cảnh báo lần đầu. Mỗi bản build mới có thể phải cấp lại quyền Accessibility.
- Windows: không điều khiển được màn hình khoá/UAC; muốn điều khiển app chạy quyền Admin thì chạy Crossdesk bằng quyền Admin.
- Linux máy chủ: vị trí con trỏ được ước lượng từ chuyển động (Wayland không cho đọc vị trí thật) — phải "đẩy" qua mép một chút mới chuyển máy. Touchpad: một ngón di chuyển, hai ngón cuộn; chạm-để-click chưa hỗ trợ khi đang điều khiển máy khác.
- Linux máy khách: kích thước màn hình lấy từ `xrandr` hoặc DRM; có thể ép bằng `CROSSDESK_SCREEN=2560x1440`.
