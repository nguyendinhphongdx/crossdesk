import { useEffect, useState } from "react"
import { Download, Github, Moon, Sun } from "lucide-react"

import { Button } from "@/components/ui/button"
import { BASE, REPO_URL, latestDownloadUrl, OS_NAME } from "@/lib/config"
import { recommendedAsset, usePlatform } from "@/lib/platform"
import { cn } from "@/lib/utils"

export function Logo({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 32 32" className={cn("size-7", className)} aria-hidden="true">
      <rect width="32" height="32" rx="8" className="fill-primary" />
      <g fill="none" stroke="white" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
        <rect x="5" y="7" width="10" height="8" rx="1.5" />
        <rect x="17" y="17" width="10" height="8" rx="1.5" />
        <path d="M15 11h4a2 2 0 0 1 2 2v4" />
      </g>
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
  if (os === "macos") return <AppleIcon className={className} />
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

export function SiteHeader({ active }: { active?: "home" | "download" }) {
  const [dark, setDark] = useTheme()
  const nav = [
    { href: `${BASE}#features`, label: "Tính năng" },
    { href: `${BASE}#how`, label: "Cách hoạt động" },
    { href: `${BASE}#security`, label: "Bảo mật" },
    { href: `${BASE}#faq`, label: "Hỏi đáp" },
  ]
  return (
    <header className="sticky top-0 z-40 border-b bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="mx-auto flex h-16 max-w-6xl items-center gap-6 px-4 sm:px-6">
        <a href={BASE} className="flex items-center gap-2.5 font-semibold tracking-tight">
          <Logo />
          <span>Share Device</span>
        </a>
        <nav className="hidden items-center gap-6 text-sm text-muted-foreground md:flex">
          {nav.map((n) => (
            <a key={n.href} href={n.href} className="transition-colors hover:text-foreground">{n.label}</a>
          ))}
        </nav>
        <div className="ml-auto flex items-center gap-1.5">
          <Button variant="ghost" size="icon" aria-label="Đổi giao diện sáng/tối" onClick={() => setDark(!dark)}>
            {dark ? <Sun /> : <Moon />}
          </Button>
          <Button variant="ghost" size="icon" asChild>
            <a href={REPO_URL} aria-label="Mã nguồn trên GitHub"><Github /></a>
          </Button>
          {active !== "download" && (
            <Button size="sm" asChild className="ml-1">
              <a href={`${BASE}download/`}><Download /> Tải về</a>
            </Button>
          )}
        </div>
      </div>
    </header>
  )
}

export function SiteFooter() {
  return (
    <footer className="border-t">
      <div className="mx-auto flex max-w-6xl flex-col gap-4 px-4 py-10 text-sm text-muted-foreground sm:flex-row sm:items-center sm:px-6">
        <div className="flex items-center gap-2.5">
          <Logo className="size-6" />
          <span>Share Device — một bàn phím, một con chuột, mọi máy tính.</span>
        </div>
        <div className="flex gap-5 sm:ml-auto">
          <a href={`${BASE}download/`} className="hover:text-foreground">Tải về</a>
          <a href={`${REPO_URL}/releases`} className="hover:text-foreground">Các phiên bản</a>
          <a href={REPO_URL} className="hover:text-foreground">GitHub</a>
          <a href={`${REPO_URL}/issues`} className="hover:text-foreground">Báo lỗi</a>
        </div>
      </div>
    </footer>
  )
}

/** Big download button pointing at the build for the visitor's OS. */
export function SmartDownloadButton({ className }: { className?: string }) {
  const platform = usePlatform()
  const asset = recommendedAsset(platform)
  if (!asset) {
    return (
      <Button size="lg" asChild className={className}>
        <a href={`${BASE}download/`}><Download /> Tải về miễn phí</a>
      </Button>
    )
  }
  return (
    <Button size="lg" asChild className={className}>
      <a href={latestDownloadUrl(asset.file)}>
        <OsIcon os={asset.os} className="size-5" /> Tải cho {OS_NAME[asset.os]}
        {asset.arch === "arm64" ? " ARM" : ""}
      </a>
    </Button>
  )
}
