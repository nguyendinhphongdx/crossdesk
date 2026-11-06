import { useEffect, useState } from "react"
import { CheckCircle2, Download as DownloadIcon, ExternalLink, FileCheck2, Info, Laptop } from "lucide-react"

import { OsIcon, SectionHeading, SiteFooter, SiteHeader } from "@/components/site"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ASSETS, OS_NAME, RELEASES_URL, latestDownloadUrl, type Asset, type OS } from "@/lib/config"
import { recommendedAsset, usePlatform } from "@/lib/platform"
import { formatSize, useLatestRelease, type Release } from "@/lib/release"

function assetInfo(release: Release | null, a: Asset) {
  const r = release?.assets.find((x) => x.name === a.file)
  return { url: r?.browser_download_url ?? latestDownloadUrl(a.file), size: r ? formatSize(r.size) : null }
}

function Code({ children }: { children: string }) {
  return <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-[0.85em]">{children}</code>
}

function Block({ children }: { children: string }) {
  return <pre className="mt-2 overflow-x-auto rounded-lg border bg-muted/60 p-3 font-mono text-xs leading-relaxed">{children}</pre>
}

const install: Record<OS, { title: string; steps: React.ReactNode[] }> = {
  macos: {
    title: "Cài trên macOS",
    steps: [
      <>Giải nén <Code>Crossdesk-macOS-universal.zip</Code> và kéo <b>Crossdesk</b> vào thư mục <b>Applications</b>.</>,
      <>Lần đầu mở: bấm chuột phải vào ứng dụng → <b>Open</b> → <b>Open</b> (ứng dụng chưa được Apple công chứng). Hoặc chạy:<Block>{"xattr -dr com.apple.quarantine \"/Applications/Crossdesk.app\""}</Block></>,
      <>Vào <b>System Settings → Privacy &amp; Security → Accessibility</b> (và <b>Input Monitoring</b>), bật <b>Crossdesk</b>, rồi mở lại ứng dụng.</>,
      <>Bảng điều khiển tự mở trong trình duyệt. Ứng dụng chạy nền, không có biểu tượng trên Dock.</>,
    ],
  },
  windows: {
    title: "Cài trên Windows",
    steps: [
      <>Chạy <Code>Crossdesk-Windows-x64.exe</Code> (hoặc bản ARM64). Không cần cài đặt.</>,
      <>Nếu SmartScreen cảnh báo: bấm <b>More info</b> → <b>Run anyway</b>.</>,
      <>Khi Windows Firewall hỏi, chọn cho phép trên mạng <b>Private</b> (cổng TCP 24800, UDP 24802).</>,
      <>Muốn điều khiển cả các cửa sổ chạy quyền Administrator thì chạy Crossdesk bằng <b>Run as administrator</b>.</>,
    ],
  },
  linux: {
    title: "Cài trên Linux",
    steps: [
      <>Giải nén và chạy script cài đặt (cần sudo một lần để cấp quyền <Code>/dev/uinput</Code> và nhóm <Code>input</Code>):<Block>{"tar xzf Crossdesk-Linux-x64.tar.gz\ncd Crossdesk-Linux-x64\nsudo ./install.sh"}</Block></>,
      <>Đăng xuất rồi đăng nhập lại để quyền nhóm <Code>input</Code> có hiệu lực.</>,
      <>Mở <b>Crossdesk</b> từ menu ứng dụng, hoặc chạy <Code>crossdesk</Code> trong terminal.</>,
      <>Chạy được trên cả X11 và Wayland. Nếu dùng tường lửa (ufw), mở cổng TCP 24800 và UDP 24802 trong mạng LAN.</>,
    ],
  },
}

export function DownloadPage() {
  const platform = usePlatform()
  const { release, failed } = useLatestRelease()
  const rec = recommendedAsset(platform)
  const [tab, setTab] = useState<OS>("macos")
  useEffect(() => {
    if (rec) setTab(rec.os)
  }, [rec?.os]) // eslint-disable-line react-hooks/exhaustive-deps

  const version = release?.tag_name
  const published = release ? new Date(release.published_at).toLocaleDateString("vi-VN") : null

  return (
    <div className="min-h-screen">
      <SiteHeader active="download" />

      <section className="relative border-b">
        <div className="bg-grid mask-radial pointer-events-none absolute inset-0 -top-16 -z-10" aria-hidden="true" />
        <div className="pointer-events-none absolute inset-x-0 -top-40 -z-10 flex justify-center" aria-hidden="true">
          <div className="h-[24rem] w-[56rem] rounded-full bg-primary/15 blur-[120px] dark:bg-primary/20" />
        </div>
        <div className="mx-auto max-w-3xl px-4 pt-16 pb-20 text-center sm:px-6 sm:pt-24">
          <div className="font-mono text-xs font-medium tracking-wider text-primary uppercase">Tải về</div>
          <h1 className="mt-3 text-4xl font-semibold tracking-[-0.03em] sm:text-6xl">Tải Crossdesk</h1>
          <p className="mx-auto mt-5 max-w-xl text-lg text-muted-foreground">
            Cài trên tất cả các máy bạn muốn dùng chung bàn phím và chuột.
            {version && <> Phiên bản <b className="text-foreground">{version}</b>{published && <> · phát hành {published}</>}.</>}
          </p>

          <Card className="mx-auto mt-10 max-w-xl py-7 text-left shadow-[0_20px_60px_-20px_rgb(0_0_0/0.25)]">
            <CardContent>
              {platform === null ? (
                <div className="h-24 animate-pulse rounded-lg bg-muted" />
              ) : rec ? (
                <div className="flex flex-col gap-5 sm:flex-row sm:items-center">
                  <div className="grid size-14 shrink-0 place-items-center rounded-xl bg-primary/10 text-primary">
                    <OsIcon os={rec.os} className="size-7" />
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <CheckCircle2 className="size-4 text-primary" /> Đã nhận diện máy của bạn
                    </div>
                    <div className="mt-1 text-lg font-semibold">{rec.label}</div>
                    <div className="text-sm text-muted-foreground">
                      {rec.detail}{assetInfo(release, rec).size && <> · {assetInfo(release, rec).size}</>}
                    </div>
                  </div>
                  <Button size="lg" asChild>
                    <a href={assetInfo(release, rec).url}><DownloadIcon /> Tải về</a>
                  </Button>
                </div>
              ) : (
                <div className="flex items-start gap-4">
                  <Laptop className="mt-1 size-6 shrink-0 text-muted-foreground" />
                  <div>
                    <div className="font-semibold">Crossdesk dành cho máy tính</div>
                    <p className="mt-1 text-sm text-muted-foreground">
                      {platform.os === "mobile"
                        ? "Bạn đang xem trên điện thoại. Hãy mở trang này trên máy tính, hoặc chọn bản phù hợp bên dưới."
                        : "Không nhận diện được hệ điều hành. Hãy chọn bản phù hợp bên dưới."}
                    </p>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
          {failed && (
            <p className="mt-4 flex items-center justify-center gap-1.5 text-xs text-muted-foreground">
              <Info className="size-3.5" /> Không tải được thông tin phiên bản — các nút vẫn trỏ tới bản mới nhất trên GitHub.
            </p>
          )}
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-4 py-24 sm:px-6">
        <SectionHeading eyebrow="Tất cả nền tảng" title="Chọn bản phù hợp với máy của bạn" />
        <div className="mt-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {ASSETS.map((a) => {
            const info = assetInfo(release, a)
            const isRec = rec?.id === a.id
            return (
              <Card key={a.id} className={isRec ? "border-primary ring-1 ring-primary/30" : ""}>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2.5">
                      <OsIcon os={a.os} className="size-5" />
                      <CardTitle>{a.label}</CardTitle>
                    </div>
                    {isRec && <Badge>Đề xuất</Badge>}
                  </div>
                  <CardDescription>{a.detail}</CardDescription>
                </CardHeader>
                <CardContent className="mt-auto flex items-center justify-between gap-3">
                  <span className="truncate font-mono text-xs text-muted-foreground" title={a.file}>{a.file}</span>
                  <Button size="sm" variant={isRec ? "default" : "outline"} asChild>
                    <a href={info.url}><DownloadIcon /> {info.size ?? "Tải"}</a>
                  </Button>
                </CardContent>
              </Card>
            )
          })}
        </div>
        <div className="mt-6 flex flex-wrap gap-x-6 gap-y-2 text-sm text-muted-foreground">
          <a href={latestDownloadUrl("SHA256SUMS.txt")} className="flex items-center gap-1.5 hover:text-foreground"><FileCheck2 className="size-4" /> SHA256 checksums</a>
          <a href={release?.html_url ?? RELEASES_URL} className="flex items-center gap-1.5 hover:text-foreground"><ExternalLink className="size-4" /> Ghi chú phát hành</a>
          <a href={RELEASES_URL} className="flex items-center gap-1.5 hover:text-foreground"><ExternalLink className="size-4" /> Các phiên bản cũ</a>
        </div>
      </section>

      <section className="border-t bg-surface">
        <div className="mx-auto max-w-3xl px-4 py-24 sm:px-6">
          <SectionHeading eyebrow="Cài đặt" title="Hướng dẫn cài đặt" />
          <Tabs value={tab} onValueChange={(v) => setTab(v as OS)} className="mt-10">
            <TabsList className="w-full sm:w-fit">
              {(["macos", "windows", "linux"] as OS[]).map((os) => (
                <TabsTrigger key={os} value={os}><OsIcon os={os} /> {OS_NAME[os]}</TabsTrigger>
              ))}
            </TabsList>
            {(["macos", "windows", "linux"] as OS[]).map((os) => (
              <TabsContent key={os} value={os}>
                <Card className="mt-2">
                  <CardHeader><CardTitle>{install[os].title}</CardTitle></CardHeader>
                  <CardContent>
                    <ol className="space-y-4 text-sm leading-relaxed">
                      {install[os].steps.map((s, i) => (
                        <li key={i} className="flex gap-3">
                          <span className="grid size-6 shrink-0 place-items-center rounded-full bg-primary/10 text-xs font-semibold text-primary">{i + 1}</span>
                          <div className="min-w-0 flex-1 pt-0.5">{s}</div>
                        </li>
                      ))}
                    </ol>
                  </CardContent>
                </Card>
              </TabsContent>
            ))}
          </Tabs>
          <p className="mt-6 text-sm text-muted-foreground">
            Sau khi cài trên các máy: đặt <b>cùng một mật khẩu nhóm</b>, chọn máy chủ, rồi kéo thả vị trí màn hình trong bảng điều khiển.
          </p>
        </div>
      </section>

      <SiteFooter />
    </div>
  )
}
