# Fangxu Kindle Transfer

[简体中文](./README.md) | **English**

[![Go 1.22+](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-3446B7.svg)](./LICENSE)
[![macOS](https://img.shields.io/badge/macOS-Intel%20%7C%20Apple%20Silicon-172039?logo=apple)](#macos)
[![Windows](https://img.shields.io/badge/Windows-x86--64-172039?logo=windows)](#windows)
[![Linux](https://img.shields.io/badge/Linux-x86--64%20%7C%20ARM64-172039?logo=linux)](#linux)

> Every book in its place, even within a small screen.

Fangxu Kindle Transfer is a free, open-source, zero-setup ebook transfer tool for Kindle. Download AZW3, MOBI, KFX, AZW, PRC, and TXT books through the Kindle built-in browser over the same Wi-Fi network, or quickly locate EPUB, PDF, DOCX, and image files on your computer and send them through Amazon Send to Kindle. It supports macOS, Windows, and Linux on Intel, Apple Silicon, x86-64, and ARM64.

Keywords: Kindle transfer, Send to Kindle, Kindle Wi-Fi transfer, Kindle browser download, ebook transfer, AZW3, MOBI, KFX, EPUB, PDF, macOS, Windows, Linux.

## ☕ Support and Follow

Fangxu Kindle Transfer will always remain free and open source. If it has saved you time or helped bring an unused Kindle back into your daily reading, consider buying the author a coffee. Every contribution encourages continued compatibility fixes, thoughtful improvements, and long-term maintenance.

You can also support the project at no cost: give it a Star, share it with another Kindle reader, or follow the **Yideng AI** WeChat official account. Thank you for helping this small tool go further.

<table>
  <tr>
    <td align="center" width="33%">
      <strong>Alipay</strong><br><br>
      <img src="https://cdn.ip21.cn/img/common/alipay-qrcode.jpg" width="180" alt="Alipay donation QR code"><br>
      <sub>Buy the author a coffee</sub>
    </td>
    <td align="center" width="33%">
      <strong>WeChat Pay</strong><br><br>
      <img src="https://cdn.ip21.cn/img/common/wechatpay-qrcode.jpg" width="180" alt="WeChat Pay donation QR code"><br>
      <sub>Support ongoing maintenance</sub>
    </td>
    <td align="center" width="33%">
      <strong>Follow Yideng AI</strong><br><br>
      <img src="https://cdn.ip21.cn/img/common/wechat-pub.png" width="180" alt="Yideng AI WeChat official account QR code"><br>
      <sub>AI tools, productivity tips, and project news</sub>
    </td>
  </tr>
</table>

## Features

- No account or database required; launch it by double-clicking
- Uses a random available port to avoid common port conflicts
- Downloads ebooks directly through the Kindle built-in browser on the same Wi-Fi network
- Copies a complete local file path and opens Send to Kindle from the desktop page
- Scans subdirectories automatically with format filters, file counts, and sorting by time or filename
- Chinese and English UI that follows the browser language and remembers manual selection
- Single-instance behavior: double-clicking again opens the existing service
- Files remain on your local network; Send to Kindle files are uploaded to Amazon only when you choose to do so

The application starts a lightweight web server on your computer. By default, it shares your Downloads folder and opens the desktop control page, where you can see the Kindle access addresses or choose another ebook folder with the native folder picker. The operating system assigns an available port automatically. If the service is already running, launching the application again opens the existing control page without starting another server. Connect the Kindle to the same Wi-Fi network and open one of the displayed addresses in its browser.

Download URLs retain the original ebook extension, such as `.mobi`, for compatibility with older Kindle browsers that determine the file type from the URL.

AZW3, MOBI, TXT, AZW, KFX, and PRC files can be downloaded directly over the local network. TXT files use a binary response type so the browser downloads them instead of displaying their content.

The Kindle page explains that other formats cannot be downloaded through the Kindle Experimental Browser and should instead be sent from the desktop page with Send to Kindle.

EPUB, PDF, DOCX, DOC, RTF, HTML/HTM, JPG/JPEG, PNG, GIF, and BMP files do not receive direct Kindle download links because the Kindle Experimental Browser rejects these extensions during download. The desktop-only **Send to Kindle from a computer browser** tab lists supported files up to 200 MB. Clicking a book copies its absolute file path. Clicking **Copy and open Send to Kindle** copies the path first and then opens Amazon's official [Send to Kindle](https://www.amazon.com/sendtokindle) page. Instructions explain how to paste a full path quickly into the Windows, macOS, and Linux file picker. Browser security prevents this application from inserting a local file into the Amazon page automatically, so you still choose the file manually after signing in.

The bookshelf is sorted by last-modified time in descending order by default, placing the newest files first. You can switch to filename sorting from the right side of the filter row and switch back at any time.

The shared directory and its subdirectories are scanned for: `AZW3`, `KFX`, `EPUB`, `AZW`, `MOBI`, `PRC`, `PDF`, `TXT`, `DOCX`, `DOC`, `RTF`, `HTML`, `HTM`, `JPG`, `JPEG`, `PNG`, `GIF`, and `BMP`.

The **Kindle built-in browser download** tab supports `AZW3, MOBI, TXT, AZW, KFX, PRC`. The desktop **Send to Kindle from a computer browser** tab lists only `PDF, DOC, DOCX, TXT, RTF, HTM, HTML, PNG, GIF, JPG, JPEG, BMP, EPUB` files no larger than 200 MB. Both tabs provide format filters ordered by common usage frequency. Each visible filter shows its file count, while formats with no files are hidden. Sorting is shared between the tabs. The responsive, full-width layout is optimized for Kindle screens and follows the Fangxu brand style.

## Download Prebuilt Releases

Visit [GitHub Releases](https://github.com/goldenwind/kindle-transfer/releases) to download a prebuilt package. Local source builds are placed in the `dist` directory by default:

| File | Operating system | Architecture |
| --- | --- | --- |
| `kindle-send-windows-amd64.exe` | Windows | x86-64 |
| `Fangxu-Kindle-Transfer-macOS.zip` | macOS application | Intel and Apple Silicon |
| `kindle-send-macos-universal` | macOS command line | Intel and Apple Silicon |
| `kindle-send-linux-amd64` | Linux | x86-64 |
| `kindle-send-linux-arm64` | Linux | ARM64, including Raspberry Pi and ARM servers |

## Launch the Application

After the application starts, it opens the desktop control page in your default browser. The page provides:

- Local-network addresses to open on the Kindle
- The current shared directory and ebook list
- A native folder selection button and a manual path input
- A separate Send to Kindle format tab
- Quick format filters for both tabs
- Sorting by newest first or by filename
- Copying the absolute path by clicking a book
- A **Copy and open Send to Kindle** button for each supported file
- A button to stop the transfer service

Only one service instance runs at a time. Launching the Windows executable, macOS application, or Linux executable again verifies the existing service and opens its control page without changing the shared directory or port.

Directory settings, local path copying, and the entire Send to Kindle tab are available only on the desktop control page. Kindle and other local-network devices see only the built-in browser download tab and directly downloadable ebooks.

### Windows

1. Double-click `kindle-send-windows-amd64.exe`.
2. Allow access to private networks if Windows Firewall asks.
3. Click **Select folder...** on the desktop page and choose your ebook directory.
4. Enter one of the displayed addresses in the Kindle browser.

Command-line usage:

```powershell
.\kindle-send-windows-amd64.exe
```

Specify an ebook directory with `--dir`:

```powershell
.\kindle-send-windows-amd64.exe --dir "D:\Books"
```

### macOS

1. Extract `Fangxu-Kindle-Transfer-macOS.zip`.
2. Move `Kindle传书.app` anywhere you like and double-click it.
3. Click **Select folder...** on the opened page and choose your ebook directory.
4. Enter one of the displayed addresses in the Kindle browser.

If macOS blocks the first launch, Control-click the application, choose **Open**, and confirm once more.

For command-line usage:

```bash
chmod +x kindle-send-macos-universal
./kindle-send-macos-universal
```

Specify an ebook directory with `--dir`:

```bash
./kindle-send-macos-universal --dir "/Users/yourname/Books"
```

### Linux

Download `kindle-send-linux-amd64` or `kindle-send-linux-arm64` for your device, then run:

```bash
chmod +x kindle-send-linux-amd64
./kindle-send-linux-amd64
```

Replace the filename with `kindle-send-linux-arm64` on ARM64 devices. The application uses `xdg-open` to open the control page. If `zenity` or `kdialog` is installed, the **Select folder** button opens a native folder picker; otherwise, enter the shared directory manually on the page.

Specify an ebook directory or fixed port:

```bash
./kindle-send-linux-amd64 --dir "/home/yourname/Books" --listen 0.0.0.0:9000
```

## Run from Source

```bash
go run . --dir "/path/to/your/ebooks"
```

Whether you use a prebuilt release or run from source, the terminal shows addresses similar to:

```text
Ebook root: /path/to/your/ebooks
Kindle access: http://192.168.1.10:49152
```

Connect the computer and Kindle to the same local network, then open the address after **Kindle access** in the Kindle browser.

Directory paths may be absolute or relative. Wrap paths containing spaces or non-ASCII characters in quotes.

The default listen address is `0.0.0.0:0`, where port `0` asks the operating system to select an available port. The port may change after the service is fully stopped and restarted, so use the current address shown on the control page or in the terminal.

To use a fixed port, specify both the directory and listen address:

```bash
./kindle-send-macos-universal --dir "/Users/me/Books" --listen 0.0.0.0:9000
```

Windows equivalent:

```powershell
.\kindle-send-windows-amd64.exe --dir "D:\Books" --listen 0.0.0.0:9000
```

Without `--dir`, the service uses the current user's Downloads directory. It falls back to the process working directory only if Downloads does not exist.

To prevent the desktop browser from opening automatically:

```bash
./kindle-send-macos-universal --no-open
```

## Build from Source

Go 1.22 or newer is required. Run the following commands in the project directory:

```bash
# Windows x86-64
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
  -trimpath -ldflags=-s -o dist/kindle-send-windows-amd64.exe .

# Linux x86-64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -trimpath -ldflags=-s -o dist/kindle-send-linux-amd64 .

# Linux ARM64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
  -trimpath -ldflags=-s -o dist/kindle-send-linux-arm64 .

# macOS Intel
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
  -trimpath -ldflags=-s -o /tmp/kindle-send-darwin-amd64 .

# macOS Apple Silicon
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build \
  -trimpath -ldflags=-s -o /tmp/kindle-send-darwin-arm64 .

# Merge the macOS Universal Binary (must run on macOS)
lipo -create -output dist/kindle-send-macos-universal \
  /tmp/kindle-send-darwin-amd64 /tmp/kindle-send-darwin-arm64
chmod +x dist/kindle-send-macos-universal
```

The macOS application wrapper is stored in `packaging/macos`. Its release archive combines the project's Universal Binary, `Info.plist`, and launch script.

## Notes

- The service has no authentication. Use it temporarily and only on a trusted home network.
- Keep the application running while transferring books. Use **Stop transfer service** on the desktop page or press `Ctrl+C` in a terminal to stop it.
- Directly supported formats may vary between Kindle models and firmware versions.
- Send to Kindle requires an internet connection and an Amazon account linked to your Kindle. Files are transferred through Amazon's service.
- Unsupported formats are never disguised with a `.mobi` extension. Renaming would only bypass a browser check and would not convert the file.

## License

Fangxu Kindle Transfer is open source under the [MIT License](./LICENSE). Issues, feature requests, and pull requests are welcome.

If this free project has been useful to you, consider supporting it through the QR codes above or following the **Yideng AI** WeChat official account.
