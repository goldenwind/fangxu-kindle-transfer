# Fangxu Kindle Transfer

[简体中文](./README.md) | **English**

[![Go 1.22+](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-3446B7.svg)](./LICENSE)
[![macOS](https://img.shields.io/badge/macOS-Intel%20%7C%20Apple%20Silicon-172039?logo=apple)](#download)
[![Windows](https://img.shields.io/badge/Windows-x86--64-172039?logo=windows)](#download)
[![Linux](https://img.shields.io/badge/Linux-x86--64%20%7C%20ARM64-172039?logo=linux)](#download)

> Every book in its place, even within a small screen.

A free, open-source, zero-setup Kindle transfer tool. Download ebooks through the Kindle built-in browser over local Wi-Fi, or quickly open Amazon Send to Kindle for other formats.

## Download

Get the appropriate package from [GitHub Releases](https://github.com/goldenwind/kindle-transfer/releases):

| System | File |
| --- | --- |
| Windows | `kindle-send-windows-amd64.exe` |
| macOS | `Fangxu-Kindle-Transfer-macOS.zip` |
| Linux x86-64 | `kindle-send-linux-amd64` |
| Linux ARM64 | `kindle-send-linux-arm64` |

## Quick Start

1. Double-click the application on your computer. Its control page opens automatically.
2. Choose your ebook folder. The Downloads folder is used by default.
3. Connect the Kindle and computer to the same Wi-Fi network.
4. Open one of the displayed local addresses in the Kindle browser, then click a book to download it.

Launching the application again opens the existing service instead of starting another one. An available port is selected automatically.

> On Windows, allow private-network access when prompted by the firewall. If macOS blocks the first launch, Control-click the app and choose **Open**. On Linux, run `chmod +x filename` before the first launch.

## Supported Formats

- **Kindle built-in browser:** `AZW3`, `MOBI`, `TXT`, `AZW`, `KFX`, `PRC`
- **Send to Kindle:** `PDF`, `DOC`, `DOCX`, `TXT`, `RTF`, `HTM`, `HTML`, `PNG`, `GIF`, `JPG`, `JPEG`, `BMP`, `EPUB`; up to 200 MB per file

On the desktop page, click a file to copy its full path. **Copy and open Send to Kindle** copies the path and opens [Amazon Send to Kindle](https://www.amazon.com/sendtokindle). Browser security still requires you to select the file manually in the upload dialog.

## Features

- Recursive folder scanning with format filters and file counts
- Newest-first sorting with an optional filename sort
- Chinese and English interface with saved language preference
- Random available port and single-instance behavior
- Responsive layout for Kindle and desktop browsers
- Direct transfers remain on the local network

## ☕ Support and Follow

Fangxu Kindle Transfer will remain free and open source. If it saves you time, consider buying the author a coffee, starring the project, or sharing it with another Kindle reader.

<table>
  <tr>
    <td align="center" width="33%"><strong>Alipay</strong><br><br><img src="https://cdn.ip21.cn/img/common/alipay-qrcode.jpg" width="160" alt="Alipay donation QR code"></td>
    <td align="center" width="33%"><strong>WeChat Pay</strong><br><br><img src="https://cdn.ip21.cn/img/common/wechatpay-qrcode.jpg" width="160" alt="WeChat Pay donation QR code"></td>
    <td align="center" width="33%"><strong>Follow Yideng AI</strong><br><br><img src="https://cdn.ip21.cn/img/common/wechat-pub.png" width="160" alt="Yideng AI WeChat account QR code"></td>
  </tr>
</table>

## Command Line

Run from source with Go 1.22 or newer:

```bash
go run . --dir "/path/to/your/ebooks"
```

Common options:

```text
--dir PATH                Set the ebook directory
--listen 0.0.0.0:9000    Use a fixed port
--no-open                 Do not open the browser at startup
```

Without `--dir`, the application uses the current user's Downloads folder. By default it listens on `0.0.0.0:0`, allowing the operating system to select an available port.

## Security

- The service has no authentication. Use it temporarily on a trusted home network only.
- Keep the application running while transferring books, then stop it from the control page.
- Send to Kindle requires an Amazon account linked to your Kindle and uploads files to Amazon.

## License

Open source under the [MIT License](./LICENSE). Issues, feature requests, and pull requests are welcome.
