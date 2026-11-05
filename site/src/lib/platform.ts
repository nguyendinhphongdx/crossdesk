import { useEffect, useState } from "react"
import { ASSETS, type Arch, type Asset, type OS } from "./config"

export interface Platform {
  os: OS | "mobile" | "unknown"
  arch: Arch | "unknown"
}

interface UAData {
  platform?: string
  mobile?: boolean
  getHighEntropyValues?: (hints: string[]) => Promise<{ architecture?: string; bitness?: string; platform?: string }>
}

function fromUserAgent(): Platform {
  const ua = navigator.userAgent
  if (/Android|iPhone|iPad|iPod/i.test(ua)) return { os: "mobile", arch: "unknown" }
  if (/Mac OS X|Macintosh/i.test(ua)) return { os: "macos", arch: "universal" }
  if (/Windows/i.test(ua)) return { os: "windows", arch: /ARM64|aarch64/i.test(ua) ? "arm64" : "x64" }
  if (/Linux|X11|CrOS/i.test(ua)) return { os: "linux", arch: /aarch64|arm64|armv8/i.test(ua) ? "arm64" : "x64" }
  return { os: "unknown", arch: "unknown" }
}

/** Detects the visitor's OS and CPU architecture (Client Hints when available). */
export function usePlatform(): Platform | null {
  const [p, setP] = useState<Platform | null>(null)
  useEffect(() => {
    const base = fromUserAgent()
    setP(base)
    const uad = (navigator as Navigator & { userAgentData?: UAData }).userAgentData
    if (!uad?.getHighEntropyValues || base.os === "mobile" || base.os === "macos") return
    uad.getHighEntropyValues(["architecture", "bitness"]).then((v) => {
      if (v.architecture === "arm" && (base.os === "windows" || base.os === "linux")) setP({ ...base, arch: "arm64" })
    }).catch(() => {})
  }, [])
  return p
}

export function recommendedAsset(p: Platform | null): Asset | undefined {
  if (!p || p.os === "mobile" || p.os === "unknown") return undefined
  return ASSETS.find((a) => a.os === p.os && (a.arch === p.arch || a.arch === "universal")) ?? ASSETS.find((a) => a.os === p.os)
}
