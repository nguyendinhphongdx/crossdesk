import {
  ArrowRight, Github, Keyboard, LayoutGrid, Lock, MonitorSmartphone, MousePointer2, Radar, Repeat, ShieldCheck, Zap,
} from "lucide-react"

import { HeroIllustration } from "@/components/hero-illustration"
import { AppleIcon, LinuxIcon, SiteFooter, SiteHeader, SmartDownloadButton, WindowsIcon } from "@/components/site"
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { BASE, REPO_URL } from "@/lib/config"
import { useLatestRelease } from "@/lib/release"

const features = [
  { icon: MousePointer2, title: "Lướt qua mép màn hình", text: "Đẩy chuột ra cạnh màn hình là sang máy bên cạnh, cả bàn phím đi theo. Không phím tắt, không bấm nút." },
  { icon: Repeat, title: "Hai chiều, ai làm chủ cũng được", text: "Dùng trackpad MacBook điều khiển Windows, hoặc bàn phím cơ của PC điều khiển Mac. Đổi máy chủ chỉ bằng một cú bấm." },
  { icon: Radar, title: "Tự tìm máy trong mạng", text: "Mở ứng dụng trên các máy, đặt cùng mật khẩu nhóm — chúng tự nhận ra nhau qua mạng LAN, không cần gõ địa chỉ IP." },
  { icon: LayoutGrid, title: "Kéo thả sắp xếp màn hình", text: "Đặt các màn hình trái, phải, trên, dưới đúng như trên bàn làm việc. Hỗ trợ nhiều máy cùng lúc." },
  { icon: Keyboard, title: "Phím tắt đúng thói quen", text: "Tự đổi Cmd ⇄ Ctrl giữa Mac và Windows/Linux: Cmd+C trên Mac thành Ctrl+C trên Windows. Bố cục phím theo vị trí, đúng mọi ngôn ngữ." },
  { icon: ShieldCheck, title: "Mã hoá toàn bộ", text: "Mọi phím bạn gõ — kể cả mật khẩu — được mã hoá AES-256-GCM trước khi đi qua mạng. Máy lạ không thể nhập nhóm." },
]

const steps = [
  { n: "1", title: "Cài trên mỗi máy", text: "Tải bản cho macOS, Windows hoặc Linux. Một file duy nhất, không driver, không tài khoản." },
  { n: "2", title: "Đặt cùng mật khẩu nhóm", text: "Các máy cùng mật khẩu tự tìm thấy nhau. Chọn máy có bàn phím và chuột bạn muốn dùng làm máy chủ." },
  { n: "3", title: "Kéo thả và dùng ngay", text: "Xếp vị trí các màn hình, rồi cứ đẩy chuột qua mép. Thế thôi." },
]

const faqs = [
  { q: "Share Device có miễn phí không?", a: "Có. Share Device miễn phí và mã nguồn mở trên GitHub. Không quảng cáo, không thu thập dữ liệu, không cần tài khoản." },
  { q: "Có cần Internet không?", a: "Không. Các máy nói chuyện trực tiếp với nhau trong mạng nội bộ (Wi-Fi hoặc dây). Không có máy chủ đám mây nào ở giữa." },
  { q: "Có an toàn khi gõ mật khẩu trên máy được điều khiển?", a: "Có. Kết nối được mã hoá bằng khoá dẫn xuất từ mật khẩu nhóm (PBKDF2 + AES-256-GCM), mỗi phiên một khoá riêng. Người khác trong cùng mạng không đọc được phím bạn gõ và không thể giả làm máy trong nhóm." },
  { q: "Dùng được trên Wayland không?", a: "Được. Trên Linux, Share Device làm việc trực tiếp với lớp input của kernel (evdev/uinput), nên chạy được cả X11 lẫn Wayland. Cần cấp quyền một lần bằng script đi kèm." },
  { q: "Vì sao macOS hỏi quyền Trợ năng (Accessibility)?", a: "Để đọc chuột/bàn phím khi Mac làm máy chủ, và để giả lập thao tác khi Mac được điều khiển. Đây là quyền bắt buộc với mọi ứng dụng chia sẻ bàn phím trên macOS." },
  { q: "Một máy chủ điều khiển được bao nhiêu máy?", a: "Không giới hạn cứng — mỗi máy là một ô trên sơ đồ, con trỏ đi xuyên từ máy này sang máy khác theo đúng vị trí bạn sắp xếp." },
  { q: "Có chia sẻ clipboard hay kéo thả file không?", a: "Chưa. Phiên bản hiện tại tập trung vào chuột và bàn phím. Clipboard đang nằm trong lộ trình phát triển." },
]

export function Landing() {
  const { release } = useLatestRelease()
  return (
    <div className="min-h-screen">
      <SiteHeader active="home" />

      {/* Hero */}
      <section className="relative overflow-hidden">
        <div className="pointer-events-none absolute inset-x-0 top-[-10rem] -z-10 flex justify-center blur-3xl" aria-hidden="true">
          <div className="aspect-[2/1] w-[60rem] bg-gradient-to-tr from-primary/25 via-primary/10 to-transparent opacity-70" />
        </div>
        <div className="mx-auto grid max-w-6xl items-center gap-12 px-4 pt-16 pb-20 sm:px-6 lg:grid-cols-[1.05fr_1fr] lg:pt-24">
          <div>
            <Badge variant="secondary" className="mb-5 gap-2 px-3 py-1 text-xs">
              <span className="size-1.5 rounded-full bg-primary" /> Miễn phí · Mã nguồn mở · {release?.tag_name ?? "Bản mới nhất"}
            </Badge>
            <h1 className="text-4xl font-semibold tracking-tight text-balance sm:text-5xl lg:text-6xl">
              Một bàn phím. Một con chuột. <span className="text-primary">Mọi máy tính</span> của bạn.
            </h1>
            <p className="mt-6 max-w-xl text-lg text-pretty text-muted-foreground">
              Share Device cho bạn dùng trackpad và bàn phím của một máy để điều khiển các máy khác trong cùng mạng —
              Mac, Windows hay Linux. Thôi cảnh với tay đổi chuột mỗi lần chuyển máy.
            </p>
            <div className="mt-8 flex flex-wrap items-center gap-3">
              <SmartDownloadButton />
              <Button size="lg" variant="outline" asChild>
                <a href={`${BASE}download/`}>Tất cả bản tải về <ArrowRight /></a>
              </Button>
            </div>
            <div className="mt-6 flex items-center gap-4 text-sm text-muted-foreground">
              <span className="flex items-center gap-1.5"><AppleIcon /> macOS 11+</span>
              <span className="flex items-center gap-1.5"><WindowsIcon /> Windows 10/11</span>
              <span className="flex items-center gap-1.5"><LinuxIcon /> Linux</span>
            </div>
          </div>
          <div className="rounded-2xl border bg-card/60 p-4 shadow-sm sm:p-6">
            <HeroIllustration />
          </div>
        </div>
      </section>

      {/* Features */}
      <section id="features" className="scroll-mt-20 border-t bg-muted/30">
        <div className="mx-auto max-w-6xl px-4 py-20 sm:px-6">
          <div className="max-w-2xl">
            <h2 className="text-3xl font-semibold tracking-tight sm:text-4xl">Mọi thứ bạn cần, không thừa thứ gì</h2>
            <p className="mt-4 text-muted-foreground">Nhẹ, nhanh, chạy nền. Không driver, không tài khoản, không đám mây.</p>
          </div>
          <div className="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {features.map((f) => (
              <Card key={f.title} className="gap-4 transition-shadow hover:shadow-md">
                <CardHeader>
                  <div className="mb-2 grid size-10 place-items-center rounded-lg bg-primary/10 text-primary">
                    <f.icon className="size-5" />
                  </div>
                  <CardTitle className="text-base">{f.title}</CardTitle>
                </CardHeader>
                <CardContent>
                  <CardDescription className="leading-relaxed">{f.text}</CardDescription>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      </section>

      {/* How it works */}
      <section id="how" className="scroll-mt-20 border-t">
        <div className="mx-auto max-w-6xl px-4 py-20 sm:px-6">
          <h2 className="text-3xl font-semibold tracking-tight sm:text-4xl">Bắt đầu trong một phút</h2>
          <ol className="mt-12 grid gap-8 md:grid-cols-3">
            {steps.map((s) => (
              <li key={s.n} className="relative">
                <div className="mb-4 grid size-10 place-items-center rounded-full border-2 border-primary text-sm font-semibold text-primary">{s.n}</div>
                <h3 className="font-semibold">{s.title}</h3>
                <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{s.text}</p>
              </li>
            ))}
          </ol>
          <Card className="mt-14 overflow-hidden p-0">
            <div className="grid md:grid-cols-2">
              <div className="p-8">
                <Badge variant="outline" className="mb-4"><MonitorSmartphone /> Bảng điều khiển</Badge>
                <h3 className="text-xl font-semibold">Thấy hết các máy, sắp xếp bằng chuột</h3>
                <p className="mt-3 text-sm leading-relaxed text-muted-foreground">
                  Ứng dụng mở một bảng điều khiển gọn gàng: danh sách máy tìm thấy trong mạng, ai đang là máy chủ,
                  ai đã kết nối — và một lưới để bạn kéo thả vị trí từng màn hình.
                </p>
              </div>
              <div className="grid place-items-center bg-muted/50 p-8">
                <div className="grid grid-cols-3 gap-2" aria-hidden="true">
                  {["", "Linux", "", "Windows", "MacBook", "Windows 2", "", "", ""].map((l, i) => (
                    <div key={i} className={`grid h-14 w-20 place-items-center rounded-md text-[11px] font-medium sm:h-16 sm:w-24 ${l ? "border-2 bg-card shadow-sm" : "bg-muted"} ${l === "MacBook" ? "border-primary text-primary" : ""}`}>
                      {l}
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </Card>
        </div>
      </section>

      {/* Security */}
      <section id="security" className="scroll-mt-20 border-t bg-muted/30">
        <div className="mx-auto grid max-w-6xl gap-12 px-4 py-20 sm:px-6 lg:grid-cols-2">
          <div>
            <h2 className="text-3xl font-semibold tracking-tight sm:text-4xl">Phím bạn gõ chỉ thuộc về bạn</h2>
            <p className="mt-4 leading-relaxed text-muted-foreground">
              Một phần mềm chia sẻ bàn phím nhìn thấy mọi thứ bạn gõ, kể cả mật khẩu. Vì vậy Share Device mã hoá từ đầu,
              không có tuỳ chọn "tắt bảo mật".
            </p>
          </div>
          <ul className="grid gap-4">
            {[
              { icon: Lock, t: "AES-256-GCM cho mọi gói tin", d: "Khoá dẫn xuất từ mật khẩu nhóm bằng PBKDF2, salt ngẫu nhiên mỗi phiên, chống phát lại và đảo thứ tự." },
              { icon: ShieldCheck, t: "Chỉ máy cùng mật khẩu mới vào nhóm", d: "Gói quảng bá được ký HMAC — máy lạ trong mạng không thể giả làm máy chủ hay đổi sơ đồ." },
              { icon: Zap, t: "Không đám mây, không theo dõi", d: "Dữ liệu chỉ đi trong mạng nội bộ của bạn. Bảng điều khiển chỉ mở được từ chính máy đó." },
            ].map((x) => (
              <li key={x.t} className="flex gap-4 rounded-xl border bg-card p-5">
                <x.icon className="mt-0.5 size-5 shrink-0 text-primary" />
                <div>
                  <div className="font-medium">{x.t}</div>
                  <div className="mt-1 text-sm text-muted-foreground">{x.d}</div>
                </div>
              </li>
            ))}
          </ul>
        </div>
      </section>

      {/* FAQ */}
      <section id="faq" className="scroll-mt-20 border-t">
        <div className="mx-auto max-w-3xl px-4 py-20 sm:px-6">
          <h2 className="text-3xl font-semibold tracking-tight sm:text-4xl">Câu hỏi thường gặp</h2>
          <Accordion type="single" collapsible className="mt-8">
            {faqs.map((f, i) => (
              <AccordionItem key={i} value={`q${i}`}>
                <AccordionTrigger className="text-base">{f.q}</AccordionTrigger>
                <AccordionContent className="leading-relaxed text-muted-foreground">{f.a}</AccordionContent>
              </AccordionItem>
            ))}
          </Accordion>
        </div>
      </section>

      {/* CTA */}
      <section className="border-t">
        <div className="mx-auto max-w-6xl px-4 py-20 sm:px-6">
          <div className="relative overflow-hidden rounded-2xl bg-primary px-8 py-14 text-center text-primary-foreground">
            <h2 className="text-3xl font-semibold tracking-tight sm:text-4xl">Dọn bớt một con chuột khỏi bàn làm việc</h2>
            <p className="mx-auto mt-4 max-w-xl opacity-85">Miễn phí cho macOS, Windows và Linux.</p>
            <div className="mt-8 flex flex-wrap justify-center gap-3">
              <SmartDownloadButton className="bg-background text-foreground hover:bg-background/90" />
              <Button size="lg" variant="ghost" asChild className="text-primary-foreground hover:bg-white/10 hover:text-primary-foreground">
                <a href={REPO_URL}><Github /> Mã nguồn</a>
              </Button>
            </div>
          </div>
        </div>
      </section>

      <SiteFooter />
    </div>
  )
}
