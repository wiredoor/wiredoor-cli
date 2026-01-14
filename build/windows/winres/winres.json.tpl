{
  "RT_GROUP_ICON": {
    "APP": {
      "0000": [
        "build/windows/icons/16x16.png",
        "build/windows/icons/24x24.png",
        "build/windows/icons/32x32.png",
        "build/windows/icons/48x48.png",
        "build/windows/icons/64x64.png",
        "build/windows/icons/128x128.png",
        "build/windows/icons/256x256.png"
      ]
    }
  },
  "RT_MANIFEST": {
    "#1": {
      "0409": {
        "identity": { "name": "net.wiredoor.cli", "version": "${VERSION}" },
        "minimum-os": "win7",
        "execution-level": "as invoker",
        "ui-access": false,
        "auto-elevate": false,
        "dpi-awareness": "per-monitor-v2",
        "disable-theming": false,
        "disable-window-filtering": false,
        "high-resolution-scrolling-aware": false,
        "ultra-high-resolution-scrolling-aware": false,
        "long-path-aware": false,
        "printer-driver-isolation": false,
        "gdi-scaling": false,
        "segment-heap": false,
        "use-common-controls-v6": false
      }
    }
  },
  "RT_VERSION": {
    "#1": {
      "0000": {
        "fixed": { "file_version": "${VERSION}.0", "product_version": "${VERSION}.0" },
        "info": {
          "0409": {
            "Comments": "No comments",
            "CompanyName": "Wiredoor",
            "FileDescription": "Wiredoor CLI Interface",
            "FileVersion": "${VERSION}",
            "InternalName": "wiredoor",
            "LegalCopyright": "Copyright (c) 2026 Wiredoor Contributors",
            "OriginalFilename": "wiredoor.exe",
            "ProductName": "Wiredoor CLI",
            "ProductVersion": "${VERSION}"
          }
        }
      }
    }
  }
}