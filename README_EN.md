# Fangxu Kindle Transfer

[简体中文](./README.md) | **English**

[![Go 1.22+](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-3446B7.svg)](./LICENSE)
[![macOS](https://img.shields.io/badge/macOS-Intel%20%7C%20Apple%20Silicon-172039?logo=apple)](#download)
[![Windows](https://img.shields.io/badge/Windows-x86--64-172039?logo=windows)](#download)
[![Linux](https://img.shields.io/badge/Linux-x86--64%20%7C%20ARM64-172039?logo=linux)](#download)

> Every book in its place, even within a small screen.

A free, open-source, zero-setup Kindle transfer tool. Download ebooks through the Kindle built-in browser over local Wi-Fi, or quickly open Amazon Send to Kindle for other formats.

![Fangxu Kindle Transfer desktop control page](./docs/images/fangxu-kindle-transfer-desktop.png)

## Download

Get the appropriate package from [GitHub Releases](https://github.com/goldenwind/fangxu-kindle-transfer/releases):

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

![Amazon Send to Kindle file upload page](./docs/images/send-to-kindle-upload.png)

On the desktop page, click a file to copy its full path. **Copy and open Send to Kindle** copies the path and opens [Amazon Send to Kindle](https://www.amazon.com/sendtokindle). Browser security still requires you to select the file manually in the upload dialog.

> **Amazon account region:** The Kindle China eBook Store closed on June 30, 2023, and its cloud-download service ended on June 30, 2024. To use Send to Kindle, mobile transfer, or email delivery, you need to register a US Amazon account and sign in to the same account on your Kindle e-reader and Kindle app.

## Features

- Recursive folder scanning with format filters and file counts
- Four sorting options in a dropdown: newest first (default), oldest first, name ascending, and name descending
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

## Mobile Transfer Notes

You can also send ebooks to your Kindle using the Kindle app on an Android phone, iPhone, or iPad:

1. Download and install the Kindle app:
   - iPhone / iPad: [App Store](https://apps.apple.com/us/app/amazon-kindle-reading-app/id302584613)
   - Android: [Google Play](https://play.google.com/store/apps/details?id=com.amazon.kindle&hl=en_US) or [APKMirror](https://www.apkmirror.com/apk/amazon-mobile-llc/amazon-kindle/)
2. Find the ebook in your device's file manager or another app, tap **Share**, and select **Kindle**.
3. Sign in to the Kindle app and your Kindle e-reader with the same Amazon account, using the same regional marketplace.
4. Wait for the library to sync, then find and download the book on your Kindle.

## Transfer by Email

Each Kindle has a dedicated Send to Kindle email address that can receive ebooks into your Kindle Library:

1. Sign in to the US Amazon [Manage Your Content and Devices](https://www.amazon.com/mycd) page, then open **Personal Document Settings** under **Preferences**.
2. Find the target device's address under **Send-to-Kindle E-Mail Settings**.
3. Add the email address you will send from to the **Approved Personal Document E-mail List**. Messages from other addresses will not be accepted.
4. Create an email, attach the ebook, and send it to the Kindle address. The subject and message body can be left blank.
5. Connect the Kindle to the internet and sync it, then find and download the ebook from your library.

An email can contain up to 25 attachments with a combined size of no more than 50 MB. Files must use one of the Send to Kindle formats listed above.

## Security

- The service has no authentication. Use it temporarily on a trusted home network only.
- Keep the application running while transferring books, then stop it from the control page.
- Send to Kindle requires an Amazon account linked to your Kindle and uploads files to Amazon.

## License

Open source under the [MIT License](./LICENSE). Issues, feature requests, and pull requests are welcome.
