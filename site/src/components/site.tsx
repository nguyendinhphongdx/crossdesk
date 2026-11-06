import { useEffect, useId, useState } from "react"
import { ArrowRight, Download, Github, Menu, Moon, Sun } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Sheet, SheetClose, SheetContent, SheetTitle, SheetTrigger } from "@/components/ui/sheet"
import { BASE, REPO_URL, latestDownloadUrl, OS_NAME } from "@/lib/config"
import { recommendedAsset, usePlatform } from "@/lib/platform"
import { cn } from "@/lib/utils"

/** Crossdesk mark: two screens as friendly eyes, the smile ends in a cursor. */
export function Logo({ className }: { className?: string }) {
  const id = useId()
  return (
    <svg viewBox="0 0 64 64" className={cn("size-7", className)} aria-hidden="true">
      <defs>
        <linearGradient id={id} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#6366F1" />
          <stop offset="1" stopColor="#A855F7" />
        </linearGradient>
      </defs>
      <rect width="64" height="64" rx="16" fill={`url(#${id})`} />
      <rect x="10" y="15" width="18" height="14" rx="4.5" fill="#fff" />
      <rect x="36" y="15" width="18" height="14" rx="4.5" fill="#fff" />
      <path d="M16 38 Q28 50 39 44" fill="none" stroke="#fff" strokeWidth="4.5" strokeLinecap="round" />
      <path d="M42 35.5 L42.6 51 L46.4 47.4 L49.4 53.2 L52.2 51.8 L49.3 46.2 L54.6 45.6 Z" fill="#fff" stroke="#fff" strokeWidth="2.2" strokeLinejoin="round" />
    </svg>
  )
}

export function AppleIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={cn("size-4", className)} aria-hidden="true">
      <path d="M16.4 12.6c0-2.4 2-3.5 2-3.6-1.1-1.6-2.8-1.8-3.4-1.8-1.4-.1-2.8.9-3.5.9s-1.9-.9-3.1-.8C6.8 7.3 5.3 8.2 4.5 9.6c-1.7 2.9-.4 7.2 1.2 9.6.8 1.2 1.8 2.5 3 2.4 1.2 0 1.7-.8 3.1-.8s1.9.8 3.1.8c1.3 0 2.1-1.2 2.9-2.4.9-1.3 1.3-2.6 1.3-2.7 0 0-2.6-1-2.7-3.9zM14.1 5.5c.7-.8 1.1-1.9 1-3-1 0-2.1.7-2.8 1.5-.6.7-1.2 1.8-1 2.9 1 .1 2.1-.6 2.8-1.4z" />
    </svg>
  )
}

export function WindowsIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={cn("size-4", className)} aria-hidden="true">
      <path d="M3 5.1 10.4 4v7.1H3V5.1zm0 13.8 7.4 1.1v-7H3v5.9zm8.2 1.2L21 21.5V13h-9.8v7.1zm0-16.2V11H21V2.5l-9.8 1.4z" />
    </svg>
  )
}

export function LinuxIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={cn("size-4", className)} aria-hidden="true">
      <path d="M12 2c-2.2 0-3.6 1.9-3.6 4.6 0 1.4-.4 2.3-1.3 3.6C5.9 11.8 5 13.6 5 15.6c0 .6.1 1.2.3 1.7-.7.4-1.3 1-1.3 1.8 0 1.3 1.4 1.6 2.8 2.1 1.2.5 2.2 1 3 .8.5-.1.9-.5 1-1 .4 0 .8.1 1.2.1s.8 0 1.2-.1c.1.5.5.9 1 1 .8.2 1.8-.3 3-.8 1.4-.5 2.8-.8 2.8-2.1 0-.8-.6-1.4-1.3-1.8.2-.5.3-1.1.3-1.7 0-2-.9-3.8-2.1-5.4-.9-1.3-1.3-2.2-1.3-3.6C15.6 3.9 14.2 2 12 2zm-1.5 4.2c.4 0 .7.5.7 1.1s-.3 1.1-.7 1.1-.7-.5-.7-1.1.3-1.1.7-1.1zm3 0c.4 0 .7.5.7 1.1s-.3 1.1-.7 1.1-.7-.5-.7-1.1.3-1.1.7-1.1zM12 9.3c.9 0 2 .5 2 .9s-1.1 1-2 1-2-.6-2-1 1.1-.9 2-.9z" />
    </svg>
  )
}

export function OsIcon({ os, className }: { os: string; className?: string }) {
  if (os === "macos" || os === "darwin") return <AppleIcon className={className} />
  if (os === "windows") return <WindowsIcon className={className} />
  return <LinuxIcon className={className} />
}

function useTheme() {
  const [dark, setDark] = useState(() => document.documentElement.classList.contains("dark"))
  useEffect(() => {
    document.documentElement.classList.toggle("dark", dark)
    try { localStorage.setItem("theme", dark ? "dark" : "light") } catch { /* ignore */ }
  }, [dark])
  return [dark, setDark] as const
}

const NAV = [
  { href: "#features", label: "Tính năng" },
  { href: "#how", label: "Cách hoạt động" },
  { href: "#security", label: "Bảo mật" },
  { href: "#platforms", label: "Nền tảng" },
  { href: "#faq", label: "Hỏi đáp" },
]

export function SiteHeader({ active }: { active?: "home" | "download" }) {
  const [dark, setDark] = useTheme()
  const [scrolled, setScrolled] = useState(false)
  useEffect(() => {
    const on = () => setScrolled(window.scrollY > 8)
    on()
    window.addEventListener("scroll", on, { passive: true })
    return () => window.removeEventListener("scroll", on)
  }, [])
  const href = (h: string) => (active === "home" ? h : `${BASE}${h}`)

  return (
    <header className={cn("sticky top-0 z-40 transition-[background-color,border-color,backdrop-filter] duration-200",
      scrolled ? "border-b bg-background/75 backdrop-blur-xl" : "border-b border-transparent")}>
      <div className="mx-auto flex h-16 max-w-6xl items-center gap-8 px-4 sm:px-6">
        <a href={BASE} className="flex items-center gap-2.5 text-[15px] font-semibold tracking-tight">
          <Logo className="size-7" />
          Crossdesk
        </a>
        <nav className="hidden items-center gap-1 text-sm md:flex" aria-label="Chính">
          {NAV.map((n) => (
            <a key={n.href} href={href(n.href)} className="rounded-md px-3 py-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground">
              {n.label}
            </a>
          ))}
        </nav>
        <div className="ml-auto flex items-center gap-1">
          <Button variant="ghost" size="icon" aria-label="Đổi giao diện sáng/tối" onClick={() => setDark(!dark)}>
            {dark ? <Sun /> : <Moon />}
          </Button>
          <Button variant="ghost" size="icon" asChild className="hidden sm:inline-flex">
            <a href={REPO_URL} aria-label="Mã nguồn trên GitHub"><Github /></a>
          </Button>
          {active !== "download" && (
            <Button size="sm" asChild className="ml-1 hidden sm:inline-flex">
              <a href={`${BASE}download/`}><Download /> Tải về</a>
            </Button>
          )}
          <Sheet>
            <SheetTrigger asChild>
              <Button variant="ghost" size="icon" className="md:hidden" aria-label="Mở menu"><Menu /></Button>
            </SheetTrigger>
            <SheetContent side="right" className="w-[85%] p-6">
              <SheetTitle className="flex items-center gap-2.5"><Logo className="size-6" /> Crossdesk</SheetTitle>
              <nav className="mt-4 flex flex-col" aria-label="Menu di động">
                {NAV.map((n) => (
                  <SheetClose asChild key={n.href}>
                    <a href={href(n.href)} className="border-b py-3.5 text-base">{n.label}</a>
                  </SheetClose>
                ))}
                <a href={REPO_URL} className="flex items-center gap-2 border-b py-3.5 text-base"><Github className="size-4" /> GitHub</a>
              </nav>
              <Button asChild className="mt-4">
                <a href={`${BASE}download/`}><Download /> Tải về miễn phí</a>
              </Button>
            </SheetContent>
          </Sheet>
        </div>
      </div>
    </header>
  )
}

export function SiteFooter() {
  const cols = [
    { title: "Sản phẩm", links: [
      { href: `${BASE}#features`, label: "Tính năng" },
      { href: `${BASE}#security`, label: "Bảo mật" },
      { href: `${BASE}#platforms`, label: "Nền tảng hỗ trợ" },
      { href: `${BASE}download/`, label: "Tải về" },
    ] },
    { title: "Tài nguyên", links: [
      { href: `${REPO_URL}#readme`, label: "Tài liệu" },
      { href: `${REPO_URL}/releases`, label: "Ghi chú phát hành" },
      { href: `${BASE}#faq`, label: "Câu hỏi thường gặp" },
    ] },
    { title: "Mã nguồn", links: [
      { href: REPO_URL, label: "GitHub" },
      { href: `${REPO_URL}/issues`, label: "Báo lỗi" },
      { href: `${REPO_URL}/issues/new`, label: "Đề xuất tính năng" },
    ] },
  ]
  return (
    <footer className="border-t bg-surface">
      <div className="mx-auto grid max-w-6xl gap-10 px-4 py-14 sm:px-6 md:grid-cols-[1.4fr_repeat(3,1fr)]">
        <div>
          <a href={BASE} className="flex items-center gap-2.5 font-semibold tracking-tight">
            <Logo className="size-6" /> Crossdesk
          </a>
          <p className="mt-4 max-w-xs text-sm leading-relaxed text-muted-foreground">
            Một bàn phím, một con chuột cho mọi máy tính trên bàn làm việc của bạn.
          </p>
          <div className="mt-5 flex items-center gap-3 text-muted-foreground">
            <AppleIcon /><WindowsIcon /><LinuxIcon />
          </div>
        </div>
        {cols.map((c) => (
          <div key={c.title}>
            <div className="text-sm font-medium">{c.title}</div>
            <ul className="mt-4 space-y-2.5 text-sm">
              {c.links.map((l) => (
                <li key={l.label}><a href={l.href} className="text-muted-foreground transition-colors hover:text-foreground">{l.label}</a></li>
              ))}
            </ul>
          </div>
        ))}
      </div>
      <div className="border-t">
        <div className="mx-auto flex max-w-6xl flex-col gap-2 px-4 py-6 text-xs text-muted-foreground sm:flex-row sm:items-center sm:justify-between sm:px-6">
          <span>© {new Date().getFullYear()} Crossdesk. Mã nguồn mở trên GitHub.</span>
          <span>Không thu thập dữ liệu · Không cần tài khoản · Hoạt động hoàn toàn trong mạng LAN</span>
        </div>
      </div>
    </footer>
  )
}

/** Primary CTA pointing at the build for the visitor's OS. */
export function SmartDownloadButton({ className, variant }: { className?: string; variant?: "default" | "outline" | "secondary" }) {
  const platform = usePlatform()
  const asset = recommendedAsset(platform)
  if (!asset) {
    return (
      <Button size="lg" variant={variant} asChild className={className}>
        <a href={`${BASE}download/`}><Download /> Tải về miễn phí</a>
      </Button>
    )
  }
  return (
    <Button size="lg" variant={variant} asChild className={className}>
      <a href={latestDownloadUrl(asset.file)}>
        <OsIcon os={asset.os} className="size-[18px]" /> Tải cho {OS_NAME[asset.os]}{asset.arch === "arm64" ? " ARM" : ""}
      </a>
    </Button>
  )
}

export function SectionHeading({ eyebrow, title, desc, center }: { eyebrow: string; title: React.ReactNode; desc?: React.ReactNode; center?: boolean }) {
  return (
    <div className={cn("max-w-2xl", center && "mx-auto text-center")}>
      <div className="font-mono text-xs font-medium tracking-wider text-primary uppercase">{eyebrow}</div>
      <h2 className="mt-3 text-3xl font-semibold tracking-tight text-balance sm:text-4xl">{title}</h2>
      {desc && <p className="mt-4 text-base leading-relaxed text-pretty text-muted-foreground sm:text-lg">{desc}</p>}
    </div>
  )
}

export function TextLink({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <a href={href} className="group inline-flex items-center gap-1 text-sm font-medium text-primary">
      {children}<ArrowRight className="size-3.5 transition-transform group-hover:translate-x-0.5" />
    </a>
  )
}
