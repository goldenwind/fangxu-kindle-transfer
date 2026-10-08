# 方序传书（Fangxu Kindle Transfer）

**简体中文** | [English](./README_EN.md)

[![Go 1.22+](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-3446B7.svg)](./LICENSE)
[![构建检查](https://github.com/goldenwind/fangxu-kindle-transfer/actions/workflows/test.yml/badge.svg)](https://github.com/goldenwind/fangxu-kindle-transfer/actions/workflows/test.yml)
[![GitHub Release](https://img.shields.io/github/v/release/goldenwind/fangxu-kindle-transfer)](https://github.com/goldenwind/fangxu-kindle-transfer/releases/latest)
[![macOS](<https://img.shields.io/badge/macOS-Intel%20%7C%20Apple%20Silicon-172039?logo=apple>)](#下载)
[![Windows](https://img.shields.io/badge/Windows-x86--64-172039?logo=windows)](#下载)
[![Linux](<https://img.shields.io/badge/Linux-x86--64%20%7C%20ARM64-172039?logo=linux>)](#下载)

> 方寸之间，自有书序。

免费、开源的 macOS、Windows、Linux Kindle 传书客户端，无需账号或会员。在同一 Wi-Fi 下，用 Kindle 内置浏览器直接下载电子书；其他格式可在电脑端快速打开 Amazon Send to Kindle 发送。

如果需要在电脑与手机、平板之间传输任意文件，请查看[方序传文件](https://github.com/goldenwind/fangxu-file-transfer)：移动端无需安装 App，扫码即可上传、下载。文档包含适用场景、项目优势，以及数据线、AirDrop、Quick Share、LocalSend、网盘等传文件方法的选择建议。

客户端采用 Tauri 2 + Rust + Vite，沿用 `fangxu-desktop-app-starter` 的桌面架构，参考 `fangxu-file-transfer` 的侧栏与服务控制界面；Go 传书服务作为内置子进程随客户端分发。电脑客户端显示连接地址、目录和服务状态；电子书列表在 Kindle 下载网页中浏览。

![v1.2.0 桌面客户端连接地址与服务控制](./docs/images/desktop-client.jpg)

## 下载

前往 [GitHub Releases](https://github.com/goldenwind/fangxu-kindle-transfer/releases) 下载对应版本：

| 系统 | 下载文件 |
| --- | --- |
| macOS 12+，Apple Silicon | 文件名包含 `aarch64` 的 `.dmg` |
| macOS 12+，Intel | 文件名包含 `x64` 的 `.dmg` |
| Windows x86-64 | `.exe` 安装程序、`.msi` 或 `windows-x64-portable.zip` 免安装版 |
| Linux x86-64 | 文件名包含 `amd64` / `x86_64` 的 `.deb` / `.AppImage` |
| Linux ARM64 | 文件名包含 `arm64` / `aarch64` 的 `.deb` / `.AppImage` |

从 **v1.2.0** 起，Release 提供原生桌面客户端安装包和 Windows 免安装 ZIP。GitHub Actions 分别构建五个系统与架构组合，全部成功后统一上传文件和 `SHA256SUMS.txt` 校验清单。历史 Release 中的 `kindle-send-*` 文件仍是原命令行版本。

**Windows 免安装版：** 下载名称包含 `windows-x64-portable.zip` 的文件，完整解压后双击 `Fangxu-Kindle-Transfer.exe` 即可启动。请保留同目录下的 `fangxu-kindle-service.exe` 和其他文件，不要直接在 ZIP 中运行，也不要只移动主程序。无需安装客户端；设置仍保存在系统应用配置目录，与安装版共享。界面依赖 WebView2 Runtime，Windows 11 和多数 Windows 10 已自带；缺少时需先安装微软的 [Evergreen Runtime](https://developer.microsoft.com/microsoft-edge/webview2/)（[分发说明](https://learn.microsoft.com/microsoft-edge/webview2/concepts/distribution)）。

macOS 最低版本与内置 Go 1.25 服务的[系统要求](https://go.dev/doc/go1.25#darwin)保持一致。

## 快速上手

1. 安装并打开“方序传书”，或解压 Windows 免安装版后双击主程序，服务立即启动；首次使用共享 Downloads 中的电子书，端口自动分配。
2. 如需更换目录或固定端口，在“传书设置”中直接选择目录、填写端口并保存，运行中的服务会自动重启并应用新设置。
3. 让 Kindle 与电脑连接同一 Wi-Fi。
4. 在 Kindle 浏览器中打开客户端显示的局域网地址，点击书名下载。
5. 传书完成后点击“停止服务”，或关闭客户端窗口。退出会同时关闭内置传书服务。

每次打开客户端都会立即启动服务，包括旧版保存过关闭自动启动的配置。服务在原生客户端初始化时启动，不依赖网页加载。目录、端口和语言保存在系统应用配置目录中，再次启动会恢复。首次使用跟随系统语言，右上角可即时切换中文、英文；共享过程中也可切换，无需停止服务或保存表单。重复打开客户端会聚焦已有窗口。运行中也可直接修改设置，保存后自动重启服务；服务已停止时，保存不会启动服务。点击启动时会保存当前表单中的设置。

Windows 首次启动时请允许防火墙访问“专用网络”。Linux 使用 AppImage 前执行 `chmod +x 文件名.AppImage`。

## 支持的格式

- **Kindle 内置浏览器下载：** `AZW3`、`MOBI`、`TXT`、`AZW`、`KFX`、`PRC`
- **Send to Kindle：** `PDF`、`DOC`、`DOCX`、`TXT`、`RTF`、`HTM`、`HTML`、`PNG`、`GIF`、`JPG`、`JPEG`、`BMP`、`EPUB`，单个文件不超过 200 MB

![Amazon Send to Kindle 文件上传页面](./docs/images/send-to-kindle-upload.png)

在客户端“电子书传输”右侧或“关于与帮助”中点击“打开 Send to Kindle”可打开 [Amazon Send to Kindle](https://www.amazon.com/sendtokindle)。可从传输页面打开电子书目录，在上传窗口中手动选择文件。

Kindle 下载网页展示可直接下载的电子书，提供搜索、格式筛选和排序；EPUB、PDF 等文件请按页面顶部提醒使用 Send to Kindle。

![Kindle 浏览器下载书架](./docs/images/desktop-bookshelf.jpg)

> **Amazon 账号地区说明：** Kindle 中国电子书店已于 2023 年 6 月 30 日停止运营，并于 2024 年 6 月 30 日停止云端下载服务。使用 Send to Kindle、移动端或 Email 传书时，需要注册美区 Amazon 账号，并在 Kindle 阅读器和 Kindle App 中登录同一个账号。

## 主要功能

- 打开客户端立即启动服务；客户端内选择目录、设置端口、保存设置、启动和停止服务
- 参考方序传文件的侧栏布局，提供电子书传输、传书设置、关于与帮助页面
- “关于与帮助”提供检查更新，按当前版本、操作系统和架构查询稳定版，显示更新说明并打开新版本下载地址（[接入说明](docs/updates.md)）
- 关闭客户端即停止共享，重复打开聚焦已有窗口
- Kindle 下载网页自动扫描子目录，支持搜索书名或子文件夹、按格式筛选并显示文件数量
- 网页下拉选择四种排序：时间从新到旧（默认）、时间从旧到新、名称正序、名称倒序
- 中文、英文界面，实时切换并记住语言选择，状态、提示和服务日志同步翻译
- [产品反馈](https://api.ip21.cn/products/11/feedback)：在客户端内展示反馈页面，提交问题或建议；登录仅用于反馈，局域网传书无需账号（[接入说明](docs/feedback.md)）
- 随机空闲端口、单实例运行
- 原生桌面客户端，保留适配 Kindle 小屏的下载网页
- 局域网直传文件不经过第三方服务器

![客户端传书设置](./docs/images/desktop-settings.jpg)

![客户端关于与帮助、检查更新](./docs/images/desktop-about.jpg)

## ☕ 支持与关注

方序传书会一直保持免费、开源。如果它帮你省下了时间，欢迎请作者喝杯咖啡、给项目一个 Star，或分享给仍在使用 Kindle 的朋友。

<table>
  <tr>
    <td align="center" width="33%"><strong>支付宝赞赏</strong><br><br><img src="https://cdn.ip21.cn/img/common/alipay-qrcode.jpg" width="160" alt="支付宝赞赏二维码"></td>
    <td align="center" width="33%"><strong>微信赞赏</strong><br><br><img src="https://cdn.ip21.cn/img/common/wechatpay-qrcode.jpg" width="160" alt="微信赞赏二维码"></td>
    <td align="center" width="33%"><strong>关注「一灯 AI」</strong><br><br><img src="https://cdn.ip21.cn/img/common/wechat-pub.png" width="160" alt="一灯 AI 微信公众号二维码"></td>
  </tr>
</table>

## 开发与打包

需要 Go 1.22+（发布构建使用 Go 1.25）、Node.js 20.19+ 或 22.12+、Rust stable，以及 [Tauri 系统依赖](https://v2.tauri.app/start/prerequisites/)。

```bash
npm ci
npm run desktop:dev       # 自动编译 Go 子进程，并启动桌面客户端
npm run check             # Go 测试/vet、前端测试/构建、Rust 协议测试
npm run desktop:build     # 构建当前平台的客户端安装包
npm run desktop:portable  # 在 Windows 构建 x64 免安装 ZIP（需要 PowerShell 7 / pwsh）
```

发布前统一修改 `package.json`、`package-lock.json`、`src-tauri/tauri.conf.json`、`src-tauri/Cargo.toml` 和 `src-tauri/Cargo.lock` 中的应用版本，运行 `npm run check:version -- v1.2.0` 和 `npm run check`。提交并推送代码后发布对应标签：

```bash
git tag -a v1.2.0 -m "方序传书 v1.2.0"
git push origin main
git push origin v1.2.0
```

标签触发 [Desktop release](https://github.com/goldenwind/fangxu-kindle-transfer/actions/workflows/release.yml)。需要重试时，可在 Actions 中手动运行该工作流，填写已经存在的标签。工作流校验标签与应用版本一致，并缓存 Rust 编译产物。

安装包输出到 `src-tauri/target/release/bundle/`。跨架构构建先安装对应 Rust target，再执行 `npm run desktop:build -- --target x86_64-apple-darwin`。Go 子进程由 `scripts/build-sidecar.mjs` 按 Tauri 目标自动编译并打包，用户无需安装 Go、Node 或 Rust。

Windows 免安装包输出到 `src-tauri/target/x86_64-pc-windows-msvc/release/bundle/portable/`。已有该目标的 release 构建时，可执行 `npm run package:windows-portable` 单独打包；发布工作流复用安装包构建产物，自动将 ZIP 上传到同一个 Release。

界面采用浅绿色主题；界面 logo、Dock 与安装包图标共用 `docs/app-icon.svg`，保留深绿底、浅绿线条的品牌配色。修改 logo 后，构建会通过 `npm run build:icons` 自动生成各平台图标。

`npm run dev` 仅预览前端，设置与服务功能需要通过 `desktop:dev` 运行。设置文件位于 Tauri `app_config_dir()/settings.json`，例如 macOS 的 `~/Library/Application Support/com.fangxu.kindle-transfer/settings.json`。

桌面客户端通过标准输入输出协议管理 Go 子进程；该控制协议不会暴露到局域网。子进程检测到客户端管道关闭时自动停止 HTTP 服务。客户端管理模式下，网页端不能修改目录或停止服务。独立命令行服务与桌面共享服务使用同一实例锁；若已有命令行服务运行，客户端会提示先关闭它。

## 命令行使用

从源码运行（需要 Go 1.22+）：

```bash
go run . --dir "/你的/电子书目录"
```

常用参数：

```text
--dir 路径               指定电子书目录
--listen 0.0.0.0:9000   使用固定端口
--no-open                启动后不自动打开浏览器
```

未指定 `--dir` 时使用当前用户的 Downloads 目录；默认监听 `0.0.0.0:0`，由系统分配空闲端口。

## 在 移动端 传书

也可以使用 Android 手机、iPhone 或 iPad 上的 Kindle App 将电子书发送到 Kindle：

1. 下载并安装 Kindle App：
   - iPhone / iPad：[App Store](https://apps.apple.com/us/app/amazon-kindle-reading-app/id302584613)
   - Android：[Google Play](https://play.google.com/store/apps/details?id=com.amazon.kindle&hl=en_US) 或 [APKMirror](https://www.apkmirror.com/apk/amazon-mobile-llc/amazon-kindle/)
2. 在系统的文件管理器或其他 App 中找到要传递的电子书，点击“分享”，然后选择 Kindle。
3. 在 Kindle App 和 Kindle 阅读器上登录同一个 Amazon 账号，并确保账号所属地区一致。
4. 等待同步完成，即可在 Kindle 书库中找到并下载这本书。

## 通过 Email 传书

每台 Kindle 都有一个专用的 Send to Kindle 邮箱，可以通过发送邮件将电子书加入 Kindle 书库：

1. 登录美区 Amazon 的[管理您的内容和设备](https://www.amazon.com/mycd)，进入“首选项”中的“个人文档设置”。
2. 在“Send to Kindle 电子邮箱设置”中找到目标 Kindle 的邮箱地址。
3. 在“已认可的个人文档电子邮箱列表”中添加你准备使用的发件邮箱，否则邮件不会被接收。
4. 新建邮件，将电子书作为附件发送到 Kindle 专用邮箱；主题和正文可以留空。
5. 将 Kindle 连接网络并同步，即可在书库中找到并下载电子书。

单封邮件最多可添加 25 个附件，附件总大小不能超过 50 MB；文件格式需符合上文“Send to Kindle”支持的格式。

## 安全提示

- 服务没有登录验证，请只在可信的家庭局域网中临时使用。
- 传书期间请保持程序运行，完成后可在客户端停止服务，或关闭客户端窗口。独立命令行版本仍可在浏览器控制页停止服务。
- Send to Kindle 需要登录与 Kindle 关联的 Amazon 账号，文件会上传至 Amazon。

## 开源许可

项目基于 [MIT License](./LICENSE) 开源，欢迎提交 Issue、功能建议和 Pull Request。
