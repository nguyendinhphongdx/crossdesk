import {
  ArrowRight, Check, Cloud, Download, FileLock2, KeyRound, Laptop, Lock, Minus, MousePointer2, Network, Package,
  Repeat, ShieldCheck, Zap,
} from "lucide-react"

import { ProductWindow } from "@/components/product-window"
import {
  AppleIcon, LinuxIcon, OsIcon, SectionHeading, SiteFooter, SiteHeader, SmartDownloadButton, TextLink, WindowsIcon,
} from "@/components/site"
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion"
import { Button } from "@/components/ui/button"
import { BASE, REPO_URL } from "@/lib/config"
import { useLatestRelease } from "@/lib/release"
import { cn } from "@/lib/utils"

/* ---------- small visuals used inside the bento cards ---------- */

function LayoutVisual() {
  const cells = [null, "linux", null, "windows", "macos", "windows", null, null, null] as const
  return (
    <div className="bg-grid grid grid-cols-3 gap-2 rounded-lg border bg-surface p-3" aria-hidden="true">
      {cells.map((c, i) => (
        <div key={i} className={cn("grid aspect-[16/10] place-items-center rounded-md",
          c ? "border-2 bg-card shadow-sm" : "border border-dashed", c === "macos" && "border-primary text-primary")}>
          {c && <OsIcon os={c} className="size-4" />}
        </div>
      ))}
    </div>
  )
}

function KeycapVisual() {
  const Key = ({ children, wide }: { children: React.ReactNode; wide?: boolean }) => (
    <kbd className={cn("inline-grid h-10 min-w-10 place-items-center rounded-md border border-b-[3px] bg-card px-2.5 font-sans text-sm font-medium shadow-sm", wide && "min-w-14")}>
      {children}
    </kbd>
  )
  return (
    <div className="flex flex-wrap items-center gap-2" aria-hidden="true">
      <div className="flex items-center gap-1.5"><Key>⌘</Key><span className="text-muted-foreground">+</span><Key>C</Key></div>
      <ArrowRight className="mx-1 size-4 text-primary" />
      <div className="flex items-center gap-1.5"><Key wide>Ctrl</Key><span className="text-muted-foreground">+</span><Key>C</Key></div>
    </div>
  )
}

function RadarVisual() {
  return (
    <div className="relative mx-auto aspect-square w-36" aria-hidden="true">
      {[1, 0.7, 0.4].map((s) => (
        <div key={s} className="absolute inset-0 m-auto rounded-full border" style={{ width: `${s * 100}%`, height: `${s * 100}%` }} />
      ))}
      <div className="radar-sweep absolute inset-0 rounded-full bg-[conic-gradient(from_0deg,transparent_0deg,color-mix(in_oklch,var(--primary)_28%,transparent)_50deg,transparent_60deg)]" />
      <div className="absolute top-1/2 left-1/2 size-2.5 -translate-1/2 rounded-full bg-primary" />
      {[["18%", "30%"], ["72%", "24%"], ["64%", "74%"]].map(([l, t], i) => (
        <span key={i} className="absolute size-2 -translate-1/2 rounded-full bg-success shadow-[0_0_0_4px_color-mix(in_oklch,var(--success)_20%,transparent)]" style={{ left: l, top: t }} />
      ))}
    </div>
  )
}

function SwapVisual() {
  const Dev = ({ os, label }: { os: string; label: string }) => (
    <div className="flex flex-col items-center gap-2">
      <div className="grid size-14 place-items-center rounded-xl border bg-card shadow-sm"><OsIcon os={os} className="size-6" /></div>
      <span className="text-xs text-muted-foreground">{label}</span>
    </div>
  )
  return (
    <div className="flex items-center justify-center gap-4" aria-hidden="true">
      <Dev os="macos" label="MacBook" />
      <div className="flex flex-col items-center gap-1 text-primary"><Repeat className="size-5" /><span className="font-mono text-[10px]">1 cú bấm</span></div>
      <Dev os="windows" label="PC" />
    </div>
  )
}

function CipherVisual() {
  return (
    <div className="w-full max-w-64 space-y-2 font-mono text-[11px]" aria-hidden="true">
      <div className="flex items-center gap-2 rounded-md border bg-card px-3 py-2"><span className="text-muted-foreground">gõ</span><span className="truncate">mật khẩu ngân hàng</span><span className="ml-auto h-3.5 w-px animate-pulse bg-foreground" /></div>
      <div className="flex justify-center text-primary"><Lock className="size-4" /></div>
      <div className="truncate rounded-md border bg-card px-3 py-2 text-muted-foreground">9f3a·c1e0·7b42·aa19·04de·5c8f·e271·b6…</div>
    </div>
  )
}

/* ---------- content ---------- */

const steps = [
  { icon: Download, title: "Cài trên mỗi máy", text: "Một file duy nhất cho macOS, Windows hoặc Linux. Không driver, không tài khoản, không khởi động lại." },
  { icon: KeyRound, title: "Đặt cùng mật khẩu nhóm", text: "Các máy cùng mật khẩu tự nhận ra nhau trong mạng LAN. Chọn máy có bàn phím bạn muốn dùng làm máy chủ." },
  { icon: MousePointer2, title: "Xếp màn hình và dùng", text: "Kéo thả vị trí các màn hình giống trên bàn làm việc, rồi cứ đẩy chuột qua mép để chuyển máy." },
]

type Cell = boolean | string
const platformRows: { label: string; mac: Cell; win: Cell; linux: Cell }[] = [
  { label: "Làm máy chủ (chia sẻ bàn phím, chuột)", mac: true, win: true, linux: true },
  { label: "Làm máy khách (được điều khiển)", mac: true, win: true, linux: true },
  { label: "Trackpad: di chuyển, cuộn mượt", mac: true, win: true, linux: "Một ngón di chuyển, hai ngón cuộn" },
  { label: "Môi trường đồ hoạ", mac: "macOS 11 trở lên", win: "Windows 10, 11", linux: "X11 và Wayland" },
  { label: "Kiến trúc CPU", mac: "Apple Silicon, Intel", win: "x64, ARM64", linux: "x64, ARM64" },
  { label: "Quyền cần cấp một lần", mac: "Accessibility", win: "Tường lửa (mạng Private)", linux: "Nhóm input, /dev/uinput" },
]

const faqs = [
  { q: "Crossdesk có miễn phí không?", a: "Có. Crossdesk miễn phí và mã nguồn mở trên GitHub. Không quảng cáo, không thu thập dữ liệu, không cần tài khoản." },
  { q: "Có cần Internet hay máy chủ đám mây không?", a: "Không. Các máy nói chuyện trực tiếp với nhau trong mạng nội bộ (Wi-Fi hoặc cáp). Không có dịch vụ trung gian nào, kể cả của chúng tôi." },
  { q: "Gõ mật khẩu trên máy được điều khiển có an toàn không?", a: "Có. Mọi phím bấm được mã hoá AES-256-GCM bằng khoá dẫn xuất từ mật khẩu nhóm (PBKDF2), mỗi phiên một khoá riêng, có chống phát lại. Người khác trong cùng mạng không đọc được và không thể giả làm máy trong nhóm." },
  { q: "Độ trễ thế nào?", a: "Sự kiện chuột và phím được gửi ngay qua TCP trong mạng LAN, không gom lô, không qua máy chủ trung gian. Độ trễ chủ yếu là độ trễ của mạng LAN, thường chỉ vài mili giây." },
  { q: "Dùng được trên Wayland không?", a: "Được. Trên Linux, Crossdesk làm việc trực tiếp với lớp input của kernel (evdev và uinput), nên chạy được trên cả X11 lẫn Wayland. Script cài đặt cấp quyền một lần." },
  { q: "Vì sao macOS yêu cầu quyền Accessibility?", a: "Để đọc chuột và bàn phím khi Mac làm máy chủ, và để thao tác khi Mac được điều khiển. Đây là quyền bắt buộc với mọi ứng dụng chia sẻ bàn phím trên macOS." },
  { q: "Một máy chủ điều khiển được bao nhiêu máy?", a: "Không giới hạn cứng. Mỗi máy là một ô trên sơ đồ; con trỏ đi xuyên từ máy này sang máy khác theo đúng vị trí bạn sắp xếp." },
  { q: "Có chia sẻ clipboard hay kéo thả file không?", a: "Chưa. Phiên bản hiện tại tập trung vào chuột và bàn phím. Clipboard nằm trong lộ trình phát triển." },
]

function PlatformCell({ v }: { v: Cell }) {
  if (v === true) return <Check className="mx-auto size-4 text-success" aria-label="Có" />
  if (v === false) return <Minus className="mx-auto size-4 text-muted-foreground" aria-label="Không" />
  return <span className="text-sm">{v}</span>
}

export function Landing() {
  const { release } = useLatestRelease()

  return (
    <div className="min-h-screen overflow-x-clip">
      <SiteHeader active="home" />

      {/* ================= Hero ================= */}
      <section className="relative">
        <div className="bg-grid mask-radial pointer-events-none absolute inset-0 -top-16 -z-10" aria-hidden="true" />
        <div className="pointer-events-none absolute inset-x-0 -top-40 -z-10 flex justify-center" aria-hidden="true">
          <div className="h-[28rem] w-[64rem] rounded-full bg-primary/15 blur-[120px] dark:bg-primary/20" />
        </div>

        <div className="mx-auto max-w-6xl px-4 pt-16 text-center sm:px-6 sm:pt-24">
          <a href={release?.html_url ?? `${REPO_URL}/releases`}
            className="inline-flex items-center gap-2 rounded-full border bg-background/60 py-1 pr-3 pl-1 text-xs backdrop-blur transition-colors hover:bg-accent">
            <span className="rounded-full bg-primary px-2 py-0.5 font-medium text-primary-foreground">Mới</span>
            <span className="text-muted-foreground">{release ? `Phiên bản ${release.tag_name} — hỗ trợ macOS, Windows, Linux` : "Hỗ trợ macOS, Windows và Linux"}</span>
            <ArrowRight className="size-3 text-muted-foreground" />
          </a>

          <h1 className="mx-auto mt-7 max-w-4xl text-[2.6rem] leading-[1.05] font-semibold tracking-[-0.035em] text-balance sm:text-6xl lg:text-7xl">
            Một bàn phím, một con chuột cho{" "}
            <span className="bg-gradient-to-r from-primary via-[oklch(0.62_0.2_300)] to-primary bg-clip-text text-transparent">mọi máy tính</span>
          </h1>
          <p className="mx-auto mt-6 max-w-2xl text-lg leading-relaxed text-pretty text-muted-foreground sm:text-xl">
            Điều khiển Mac, Windows và Linux bằng một bộ bàn phím, chuột duy nhất. Đẩy con trỏ qua mép màn hình là sang máy bên cạnh — mã hoá toàn bộ, không cần Internet.
          </p>

          <div className="mt-9 flex flex-col items-center justify-center gap-3 sm:flex-row">
            <SmartDownloadButton className="h-12 w-full px-7 text-[15px] sm:w-auto" />
            <Button size="lg" variant="outline" asChild className="h-12 w-full px-6 text-[15px] sm:w-auto">
              <a href={`${BASE}download/`}>Xem tất cả bản tải về</a>
            </Button>
          </div>
          <p className="mt-4 text-xs text-muted-foreground">Miễn phí · Mã nguồn mở · Không cần tài khoản</p>
        </div>

        <div className="mx-auto mt-16 max-w-5xl px-4 sm:mt-20 sm:px-6">
          <ProductWindow />
        </div>
      </section>

      {/* ================= Trust strip ================= */}
      <section className="mt-20 border-y bg-surface">
        <div className="mx-auto grid max-w-6xl grid-cols-2 divide-border px-4 sm:px-6 md:grid-cols-4 md:divide-x">
          {[
            { icon: Laptop, k: "3 hệ điều hành", v: "macOS · Windows · Linux" },
            { icon: FileLock2, k: "AES-256-GCM", v: "Mã hoá mọi phím bấm" },
            { icon: Cloud, k: "0 máy chủ đám mây", v: "Chạy hoàn toàn trong LAN" },
            { icon: Package, k: "1 file duy nhất", v: "Không driver, không cài đặt rườm rà" },
          ].map((s) => (
            <div key={s.k} className="flex items-start gap-3 px-2 py-7 md:px-6">
              <s.icon className="mt-0.5 size-5 shrink-0 text-primary" />
              <div>
                <div className="text-sm font-semibold">{s.k}</div>
                <div className="mt-0.5 text-xs text-muted-foreground">{s.v}</div>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* ================= Features (bento) ================= */}
      <section id="features" className="scroll-mt-20">
        <div className="mx-auto max-w-6xl px-4 py-24 sm:px-6 sm:py-32">
          <SectionHeading eyebrow="Tính năng" title="Làm đúng một việc, và làm thật tốt"
            desc="Không phím tắt khó nhớ, không cấu hình IP. Mở ứng dụng, đặt mật khẩu, đẩy chuột qua mép màn hình." />

          <div className="mt-14 grid gap-4 md:grid-cols-6">
            <BentoCard className="md:col-span-4" title="Sắp xếp màn hình bằng kéo thả"
              text="Đặt các máy trái, phải, trên, dưới đúng như trên bàn. Con trỏ đi xuyên nhiều máy theo sơ đồ bạn vẽ.">
              <div className="mx-auto w-full max-w-sm"><LayoutVisual /></div>
            </BentoCard>
            <BentoCard className="md:col-span-2" title="Tự tìm máy trong mạng"
              text="Các máy cùng mật khẩu nhóm tự nhận ra nhau. Không cần nhập địa chỉ IP.">
              <RadarVisual />
            </BentoCard>
            <BentoCard className="md:col-span-2" title="Hai chiều, ai làm chủ cũng được"
              text="Trackpad MacBook điều khiển PC, hoặc bàn phím cơ của PC điều khiển Mac.">
              <SwapVisual />
            </BentoCard>
            <BentoCard className="md:col-span-2" title="Phím tắt đúng thói quen"
              text="Tự đổi Cmd ⇄ Ctrl giữa Mac và Windows, Linux. Phím theo vị trí, đúng mọi bố cục.">
              <KeycapVisual />
            </BentoCard>
            <BentoCard className="md:col-span-2" title="Mã hoá từng phím bấm"
              text="Kể cả mật khẩu bạn gõ cũng không bao giờ đi qua mạng dưới dạng đọc được.">
              <CipherVisual />
            </BentoCard>
          </div>
        </div>
      </section>

      {/* ================= How it works ================= */}
      <section id="how" className="scroll-mt-20 border-t bg-surface">
        <div className="mx-auto max-w-6xl px-4 py-24 sm:px-6 sm:py-32">
          <SectionHeading eyebrow="Cách hoạt động" title="Sẵn sàng trong chưa đầy một phút" />
          <ol className="relative mt-14 grid gap-10 md:grid-cols-3 md:gap-8">
            <div className="absolute top-5 right-[16%] left-[16%] hidden h-px bg-gradient-to-r from-transparent via-border to-transparent md:block" aria-hidden="true" />
            {steps.map((s, i) => (
              <li key={s.title} className="relative">
                <div className="relative grid size-10 place-items-center rounded-xl border bg-card shadow-sm">
                  <s.icon className="size-[18px] text-primary" />
                </div>
                <div className="mt-5 font-mono text-xs text-muted-foreground">Bước {i + 1}</div>
                <h3 className="mt-1 text-lg font-semibold tracking-tight">{s.title}</h3>
                <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{s.text}</p>
              </li>
            ))}
          </ol>
        </div>
      </section>

      {/* ================= Security ================= */}
      <section id="security" className="scroll-mt-20 border-t">
        <div className="mx-auto grid max-w-6xl gap-14 px-4 py-24 sm:px-6 sm:py-32 lg:grid-cols-[1fr_1.1fr] lg:items-center">
          <div>
            <SectionHeading eyebrow="Bảo mật" title="Phím bạn gõ chỉ thuộc về bạn"
              desc="Phần mềm chia sẻ bàn phím nhìn thấy mọi thứ bạn gõ — kể cả mật khẩu. Vì vậy mã hoá được bật sẵn và không có cách nào tắt." />
            <ul className="mt-8 space-y-5">
              {[
                { icon: Lock, t: "AES-256-GCM cho mọi gói tin", d: "Khoá dẫn xuất bằng PBKDF2 với salt ngẫu nhiên mỗi phiên. Bộ đếm nonce chống phát lại và đảo thứ tự." },
                { icon: ShieldCheck, t: "Chỉ máy cùng mật khẩu mới vào nhóm", d: "Gói quảng bá được ký HMAC-SHA256 — máy lạ không thể giả làm máy chủ hay sửa sơ đồ." },
                { icon: Network, t: "Bảng điều khiển chỉ mở từ chính máy đó", d: "Giao diện chỉ lắng nghe 127.0.0.1, có token phiên và chống DNS rebinding." },
              ].map((x) => (
                <li key={x.t} className="flex gap-4">
                  <div className="grid size-9 shrink-0 place-items-center rounded-lg border bg-card"><x.icon className="size-4 text-primary" /></div>
                  <div>
                    <div className="font-medium">{x.t}</div>
                    <div className="mt-1 text-sm leading-relaxed text-muted-foreground">{x.d}</div>
                  </div>
                </li>
              ))}
            </ul>
          </div>

          {/* protocol diagram */}
          <div className="rounded-2xl border bg-surface p-6 sm:p-8">
            <div className="font-mono text-xs text-muted-foreground">Phiên kết nối</div>
            <div className="mt-6 grid grid-cols-[1fr_auto_1fr] items-start gap-x-3 gap-y-4 text-sm">
              <div className="rounded-lg border bg-card p-3 text-center font-medium"><OsIcon os="macos" className="mx-auto mb-1.5 size-5" />Máy chủ</div>
              <div />
              <div className="rounded-lg border bg-card p-3 text-center font-medium"><OsIcon os="windows" className="mx-auto mb-1.5 size-5" />Máy khách</div>
              {[
                { dir: "→", label: "UDP · quảng bá có chữ ký HMAC", note: "Tìm máy cùng nhóm" },
                { dir: "←", label: "TCP · kết nối tới cổng 24800", note: "Máy khách chủ động nối" },
                { dir: "→", label: "Salt ngẫu nhiên 128-bit", note: "PBKDF2 → khoá AES-256" },
                { dir: "⇄", label: "Khung AES-256-GCM", note: "Chuột, phím, cuộn — có xác thực" },
              ].map((r, i) => (
                <div key={i} className="col-span-3 grid grid-cols-[1fr_auto_1fr] items-center gap-3">
                  <div className="h-px bg-border" />
                  <div className="rounded-md border bg-card px-3 py-2 text-center">
                    <div className="font-mono text-[11px] font-medium"><span className="mr-1.5 text-primary">{r.dir}</span>{r.label}</div>
                    <div className="mt-0.5 text-[11px] text-muted-foreground">{r.note}</div>
                  </div>
                  <div className="h-px bg-border" />
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* ================= Platforms ================= */}
      <section id="platforms" className="scroll-mt-20 border-t bg-surface">
        <div className="mx-auto max-w-6xl px-4 py-24 sm:px-6 sm:py-32">
          <SectionHeading eyebrow="Nền tảng" title="Một ứng dụng, ba hệ điều hành"
            desc="Cùng một giao thức, cùng một giao diện điều khiển. Trộn máy thoải mái trên bàn làm việc." />
          <div className="mt-12 overflow-x-auto rounded-xl border bg-card">
            <table className="w-full min-w-[640px] text-left">
              <thead>
                <tr className="border-b">
                  <th className="w-[34%] px-5 py-4 text-sm font-medium text-muted-foreground"></th>
                  {[{ os: "macos", n: "macOS" }, { os: "windows", n: "Windows" }, { os: "linux", n: "Linux" }].map((h) => (
                    <th key={h.os} className="px-5 py-4 text-center text-sm font-semibold">
                      <span className="inline-flex items-center gap-2"><OsIcon os={h.os} /> {h.n}</span>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {platformRows.map((r) => (
                  <tr key={r.label} className="border-b last:border-0">
                    <td className="px-5 py-4 text-sm font-medium">{r.label}</td>
                    <td className="px-5 py-4 text-center"><PlatformCell v={r.mac} /></td>
                    <td className="px-5 py-4 text-center"><PlatformCell v={r.win} /></td>
                    <td className="px-5 py-4 text-center"><PlatformCell v={r.linux} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="mt-6"><TextLink href={`${BASE}download/`}>Xem hướng dẫn cài đặt cho từng hệ điều hành</TextLink></div>
        </div>
      </section>

      {/* ================= FAQ ================= */}
      <section id="faq" className="scroll-mt-20 border-t">
        <div className="mx-auto grid max-w-6xl gap-12 px-4 py-24 sm:px-6 sm:py-32 lg:grid-cols-[1fr_1.6fr]">
          <div>
            <SectionHeading eyebrow="Hỏi đáp" title="Câu hỏi thường gặp" />
            <p className="mt-4 text-sm text-muted-foreground">
              Không thấy câu trả lời? <a href={`${REPO_URL}/issues`} className="font-medium text-primary hover:underline">Hỏi trên GitHub</a>.
            </p>
          </div>
          <Accordion type="single" collapsible defaultValue="q0" className="rounded-xl border bg-card px-5">
            {faqs.map((f, i) => (
              <AccordionItem key={i} value={`q${i}`}>
                <AccordionTrigger className="py-5 text-[15px] hover:no-underline">{f.q}</AccordionTrigger>
                <AccordionContent className="pb-5 leading-relaxed text-muted-foreground">{f.a}</AccordionContent>
              </AccordionItem>
            ))}
          </Accordion>
        </div>
      </section>

      {/* ================= CTA ================= */}
      <section className="border-t">
        <div className="mx-auto max-w-6xl px-4 py-24 sm:px-6">
          <div className="relative overflow-hidden rounded-3xl border bg-card px-6 py-16 text-center sm:px-12 sm:py-20">
            <div className="bg-grid mask-radial pointer-events-none absolute inset-0" aria-hidden="true" />
            <div className="pointer-events-none absolute -top-24 left-1/2 h-64 w-[40rem] -translate-x-1/2 rounded-full bg-primary/20 blur-[100px]" aria-hidden="true" />
            <div className="relative">
              <Zap className="mx-auto size-8 text-primary" />
              <h2 className="mx-auto mt-5 max-w-2xl text-3xl font-semibold tracking-tight text-balance sm:text-5xl">
                Dọn bớt một con chuột khỏi bàn làm việc
              </h2>
              <p className="mx-auto mt-5 max-w-lg text-muted-foreground sm:text-lg">Miễn phí cho macOS, Windows và Linux. Cài đặt trong một phút.</p>
              <div className="mt-9 flex flex-col items-center justify-center gap-3 sm:flex-row">
                <SmartDownloadButton className="h-12 w-full px-7 sm:w-auto" />
                <Button size="lg" variant="outline" asChild className="h-12 w-full px-6 sm:w-auto">
                  <a href={REPO_URL}>Xem mã nguồn</a>
                </Button>
              </div>
              <div className="mt-8 flex items-center justify-center gap-5 text-muted-foreground">
                <AppleIcon className="size-5" /><WindowsIcon className="size-5" /><LinuxIcon className="size-5" />
              </div>
            </div>
          </div>
        </div>
      </section>

      <SiteFooter />
    </div>
  )
}

function BentoCard({ title, text, className, children }: { title: string; text: string; className?: string; children: React.ReactNode }) {
  return (
    <div className={cn("group flex flex-col overflow-hidden rounded-2xl border bg-card transition-colors hover:border-primary/30", className)}>
      <div className="flex min-h-48 flex-1 items-center justify-center border-b bg-surface p-6 sm:p-8">{children}</div>
      <div className="p-6">
        <h3 className="font-semibold tracking-tight">{title}</h3>
        <p className="mt-1.5 text-sm leading-relaxed text-muted-foreground">{text}</p>
      </div>
    </div>
  )
}

