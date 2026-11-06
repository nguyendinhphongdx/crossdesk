// The GitHub repository that hosts the releases. Override at build time with
// VITE_GITHUB_REPO=owner/name (the Pages workflow sets it automatically).
export const REPO: string = import.meta.env.VITE_GITHUB_REPO || "nguyendinhphongdx/crossdesk"
export const REPO_URL = `https://github.com/${REPO}`
export const RELEASES_URL = `${REPO_URL}/releases`
export const BASE = import.meta.env.BASE_URL

export type OS = "macos" | "windows" | "linux"
export type Arch = "x64" | "arm64" | "universal"

export interface Asset {
  id: string
  os: OS
  arch: Arch
  file: string
  label: string
  detail: string
}

// Release asset names produced by .github/workflows/release.yml.
export const ASSETS: Asset[] = [
  { id: "mac", os: "macos", arch: "universal", file: "Crossdesk-macOS-universal.zip", label: "macOS", detail: "Apple Silicon & Intel · macOS 11+" },
  { id: "win-x64", os: "windows", arch: "x64", file: "Crossdesk-Windows-x64.exe", label: "Windows", detail: "x64 · Windows 10 / 11" },
  { id: "win-arm64", os: "windows", arch: "arm64", file: "Crossdesk-Windows-arm64.exe", label: "Windows ARM", detail: "ARM64 · Snapdragon / Surface" },
  { id: "linux-x64", os: "linux", arch: "x64", file: "Crossdesk-Linux-x64.tar.gz", label: "Linux", detail: "x64 · X11 & Wayland" },
  { id: "linux-arm64", os: "linux", arch: "arm64", file: "Crossdesk-Linux-arm64.tar.gz", label: "Linux ARM", detail: "ARM64 · Raspberry Pi 4/5, ARM laptops" },
]

export const latestDownloadUrl = (file: string) => `${RELEASES_URL}/latest/download/${file}`

export const OS_NAME: Record<OS, string> = { macos: "macOS", windows: "Windows", linux: "Linux" }
