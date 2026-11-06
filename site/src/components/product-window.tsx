import { useEffect, useState } from "react"
import { Keyboard, Lock, MousePointer2, Wifi } from "lucide-react"

import { OsIcon } from "@/components/site"
import { cn } from "@/lib/utils"

const DEVICES = [
  { id: "linux", name: "ubuntu-workstation", os: "linux", res: "2560×1440", role: "Máy khách" },
  { id: "mac", name: "MacBook Pro", os: "macos", res: "1512×982", role: "Máy chủ" },
  { id: "win", name: "DESKTOP-WIN11", os: "windows", res: "1920×1080", role: "Máy khách" },
] as const

// Order the cursor visits the screens in (matches the hero animation).
const TOUR = ["mac", "linux", "mac", "win"] as const

/** A faithful, animated rendering of the app's control panel. */
export function ProductWindow() {
  const [step, setStep] = useState(0)
  useEffect(() => {
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return
    const t = setInterval(() => setStep((s) => (s + 1) % TOUR.length), 2200)
    return () => clearInterval(t)
  }, [])
  const active = TOUR[step]
  const activeDev = DEVICES.find((d) => d.id === active)!

  return (
    <div className="relative">
      <div className="absolute -inset-px rounded-[14px] bg-gradient-to-b from-primary/40 via-border to-border opacity-80" aria-hidden="true" />
      <div className="relative overflow-hidden rounded-xl bg-card shadow-[0_30px_80px_-20px_rgb(0_0_0/0.35)] dark:shadow-[0_30px_80px_-20px_rgb(0_0_0/0.8)]">
        {/* window chrome */}
        <div className="flex h-10 items-center gap-2 border-b bg-surface px-4">
          <span className="size-3 rounded-full bg-[#ff5f57]" />
          <span className="size-3 rounded-full bg-[#febc2e]" />
          <span className="size-3 rounded-full bg-[#28c840]" />
          <div className="mx-auto flex items-center gap-1.5 rounded-md border bg-background px-3 py-1 font-mono text-[11px] text-muted-foreground">
            <Lock className="size-3" /> 127.0.0.1:24801
          </div>
          <span className="w-[52px]" />
        </div>

        <div className="grid md:grid-cols-[260px_1fr]">
          {/* devices */}
          <aside className="border-b p-4 md:border-r md:border-b-0">
            <div className="flex items-center justify-between">
              <span className="font-mono text-[10px] font-medium tracking-wider text-muted-foreground uppercase">Thiết bị</span>
              <span className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
                <span className="relative flex size-1.5">
                  <span className="ping-soft absolute inline-flex size-full rounded-full bg-success" />
                  <span className="relative inline-flex size-1.5 rounded-full bg-success" />
                </span>
                Đang quét LAN
              </span>
            </div>
            <ul className="mt-3 space-y-1.5">
              {DEVICES.map((d) => (
                <li key={d.id} className={cn("flex items-center gap-3 rounded-lg border px-3 py-2.5 transition-colors duration-500",
                  d.id === active ? "border-primary/50 bg-primary/5" : "border-transparent")}>
                  <div className="grid size-8 shrink-0 place-items-center rounded-md bg-muted"><OsIcon os={d.os} className="size-4" /></div>
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-[13px] font-medium">{d.name}</div>
                    <div className="mt-0.5 flex items-center gap-1.5 text-[11px] text-muted-foreground">
                      <span className={cn("rounded px-1.5 py-px text-[10px] font-medium",
                        d.role === "Máy chủ" ? "bg-primary/12 text-primary" : "bg-success/12 text-success")}>
                        {d.role}
                      </span>
                      {d.res}
                    </div>
                  </div>
                </li>
              ))}
            </ul>
            <div className="mt-4 hidden space-y-2 rounded-lg border bg-surface p-3 text-[11px] text-muted-foreground md:block">
              <div className="flex items-center gap-2"><Lock className="size-3.5 text-primary" /> Mã hoá AES-256-GCM</div>
              <div className="flex items-center gap-2"><Wifi className="size-3.5 text-primary" /> 3 máy trong mạng 192.168.1.0/24</div>
            </div>
          </aside>

          {/* layout */}
          <div className="relative p-5 sm:p-6">
            <div className="flex items-center justify-between">
              <span className="font-mono text-[10px] font-medium tracking-wider text-muted-foreground uppercase">Sắp xếp màn hình</span>
              <span className="flex items-center gap-1.5 rounded-md border bg-background px-2 py-1 text-[11px]">
                <MousePointer2 className="size-3 text-primary" /> Con trỏ: <b className="font-medium">{activeDev.name}</b>
              </span>
            </div>
            <div className="bg-grid mt-4 rounded-lg border bg-surface p-4 sm:p-6">
              <div className="grid grid-cols-3 items-center gap-3 sm:gap-4">
                {DEVICES.map((d) => {
                  const on = d.id === active
                  return (
                    <div key={d.id} className={cn(
                      "relative flex aspect-[16/10] flex-col items-center justify-center gap-1 rounded-md border-2 bg-card text-center shadow-sm transition-all duration-500",
                      on ? "border-primary shadow-[0_0_0_4px_color-mix(in_oklch,var(--primary)_15%,transparent)]" : "border-border",
                    )}>
                      <OsIcon os={d.os} className={cn("size-4 sm:size-5", on ? "text-primary" : "text-muted-foreground")} />
                      <div className="max-w-full truncate px-1 text-[10px] font-medium sm:text-xs">{d.name}</div>
                      {on && (
                        <MousePointer2 className="absolute right-[18%] bottom-[16%] size-4 fill-foreground text-background drop-shadow sm:size-5" />
                      )}
                    </div>
                  )
                })}
              </div>
              <div className="mt-4 grid grid-cols-3 gap-3 opacity-60 sm:gap-4" aria-hidden="true">
                {[0, 1, 2].map((i) => <div key={i} className="aspect-[16/10] rounded-md border border-dashed" />)}
              </div>
            </div>
            <p className="mt-4 text-[11px] leading-relaxed text-muted-foreground">
              <Keyboard className="mr-1.5 inline size-3.5 align-[-3px]" />
              {active === "mac" ? (
                <>Bàn phím &amp; trackpad đang dùng trên chính <b className="font-medium text-foreground">MacBook Pro</b></>
              ) : (
                <>Bàn phím &amp; trackpad của <b className="font-medium text-foreground">MacBook Pro</b> đang điều khiển <b className="font-medium text-foreground">{activeDev.name}</b></>
              )}
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
