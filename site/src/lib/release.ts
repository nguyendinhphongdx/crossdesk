import { useEffect, useState } from "react"
import { REPO } from "./config"

export interface ReleaseAsset { name: string; size: number; browser_download_url: string; download_count: number }
export interface Release { tag_name: string; name: string; html_url: string; published_at: string; assets: ReleaseAsset[] }

/** Latest GitHub release (cached for 10 minutes; null while loading or if unavailable). */
export function useLatestRelease() {
  const [release, setRelease] = useState<Release | null>(null)
  const [failed, setFailed] = useState(false)
  useEffect(() => {
    const key = `release:${REPO}`
    try {
      const cached = JSON.parse(sessionStorage.getItem(key) || "null")
      if (cached && Date.now() - cached.at < 10 * 60 * 1000) { setRelease(cached.data); return }
    } catch { /* storage unavailable */ }
    fetch(`https://api.github.com/repos/${REPO}/releases/latest`, { headers: { Accept: "application/vnd.github+json" } })
      .then((r) => (r.ok ? r.json() : Promise.reject(new Error(String(r.status)))))
      .then((data: Release) => {
        setRelease(data)
        try { sessionStorage.setItem(key, JSON.stringify({ at: Date.now(), data })) } catch { /* ignore */ }
      })
      .catch(() => setFailed(true))
  }, [])
  return { release, failed }
}

export function formatSize(bytes: number) {
  return bytes > 1024 * 1024 ? `${(bytes / 1024 / 1024).toFixed(1)} MB` : `${Math.round(bytes / 1024)} KB`
}
