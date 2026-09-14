# 方序传书（Fangxu Kindle Transfer）

**简体中文** | [English](./README_EN.md)

[![Go 1.22+](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-3446B7.svg)](./LICENSE)
[![macOS](https://img.shields.io/badge/macOS-Intel%20%7C%20Apple%20Silicon-172039?logo=apple)](#macos)
[![Windows](https://img.shields.io/badge/Windows-x86--64-172039?logo=windows)](#windows)
[![Linux](https://img.shields.io/badge/Linux-x86--64%20%7C%20ARM64-172039?logo=linux)](#linux)

> 方寸之间，自有书序。

方序传书是一款免费、开源、无需安装的 Kindle 局域网传书工具。它支持在同一 Wi-Fi 下通过 Kindle 内置浏览器下载 AZW3、MOBI、KFX、AZW、PRC、TXT 电子书，也能在电脑端快速定位 EPUB、PDF、DOCX 和图片文件并通过 Amazon Send to Kindle 云端传书，兼容 macOS、Windows、Linux、Intel、Apple Silicon 与 ARM64。

关键词：Kindle 传书、Kindle Transfer、Send to Kindle、Kindle 局域网传书、Kindle 浏览器下载、AZW3、MOBI、KFX、EPUB、PDF、macOS、Windows、Linux。

## ☕ 支持与关注

方序传书会一直保持免费、开源。如果它帮你省下了折腾传书工具的时间，或让闲置的 Kindle 再次回到手边，欢迎请作者喝杯咖啡。每一份支持，都会成为继续修复兼容性、打磨体验和维护项目的动力。

不方便赞赏也没关系：给项目一个 Star、分享给仍在使用 Kindle 的朋友，或关注微信公众号「一灯 AI」，同样是在帮助这个小工具走得更远。感谢你的使用与认可。

<table>
  <tr>
    <td align="center" width="33%">
      <strong>支付宝赞赏</strong><br><br>
      <img src="https://cdn.ip21.cn/img/common/alipay-qrcode.jpg" width="180" alt="支付宝赞赏二维码"><br>
      <sub>请作者喝杯咖啡</sub>
    </td>
    <td align="center" width="33%">
      <strong>微信赞赏</strong><br><br>
      <img src="https://cdn.ip21.cn/img/common/wechatpay-qrcode.jpg" width="180" alt="微信赞赏二维码"><br>
      <sub>支持免费项目持续维护</sub>
    </td>
    <td align="center" width="33%">
      <strong>关注「一灯 AI」</strong><br><br>
      <img src="https://cdn.ip21.cn/img/common/wechat-pub.png" width="180" alt="一灯 AI 微信公众号二维码"><br>
      <sub>获取 AI 工具、效率技巧与项目动态</sub>
    </td>
  </tr>
</table>

## 核心特性

- 无需账号或数据库，双击即可在本机启动
- 随机选择空闲端口，避免常见端口冲突
- 同一 Wi-Fi 内通过 Kindle 内置浏览器直接下载电子书
- 电脑端一键复制完整文件路径并打开 Send to Kindle
- 自动扫描子目录，支持格式筛选、数量统计、时间或文件名排序
- 中文与英文界面，可自动跟随浏览器语言并记住手动选择
- 单实例运行，再次双击直接打开已有服务
- 文件仅在局域网中流转；Send to Kindle 文件由用户主动上传至 Amazon

在电脑上启动一个轻量 Web 服务器。双击程序后默认共享用户的“下载/Downloads”目录，并自动打开电脑端控制页，显示 Kindle 访问地址，也可通过系统文件夹选择器更换电子书目录。默认由系统随机分配一个空闲端口，避免与其他程序冲突；如果服务已经运行，再次双击会直接打开原来的控制页，而不会重复启动。Kindle 连接同一 Wi-Fi 后，用浏览器打开地址即可浏览书架。

下载地址会直接以电子书扩展名结尾（例如 `.mobi`），兼容只根据 URL 判断文件类型的老款 Kindle 浏览器。

AZW3、MOBI、TXT、AZW、KFX 和 PRC 提供局域网直接下载。TXT 使用二进制响应类型强制触发下载，避免被浏览器直接打开。

页面会明确提示：其他格式无法在 Kindle 体验版浏览器中直接下载，请在电脑端打开同一页面，通过电脑浏览器 Send to Kindle 云端传书。

EPUB、PDF、DOCX、DOC、RTF、HTML/HTM、JPG/JPEG、PNG、GIF 和 BMP 不生成直接下载链接，因为 Kindle“体验版网页浏览器”会在下载阶段拒绝这些扩展名。电脑端独立的“电脑浏览器 Send to Kindle 云端传书”标签页仅列出 Send to Kindle 支持且不超过 200 MB 的文件。点击书籍条目会复制该文件的绝对路径；点击“复制并打开Send to Kindle”会先复制路径，再前往亚马逊官方 [Send to Kindle](https://www.amazon.com/sendtokindle) 上传页。页面同时提供 Windows 与 macOS 文件选择窗口快速粘贴完整路径的说明。登录亚马逊账号并手动选择对应文件即可；受浏览器安全机制限制，本程序不能自动把本地文件填入亚马逊网页。

书架默认按文件最后修改时间倒序排列，最新文件在最前面。标题右侧可切换为按文件名排序，也可随时切回时间排序。

支持扫描共享目录及其子目录中的：`AZW3`、`KFX`、`EPUB`、`AZW`、`MOBI`、`PRC`、`PDF`、`TXT`、`DOCX`、`DOC`、`RTF`、`HTML`、`HTM`、`JPG`、`JPEG`、`PNG`、`GIF`、`BMP`。

“Kindle 内置浏览器下载”标签页支持 `AZW3、MOBI、TXT、AZW、KFX、PRC`；电脑端“电脑浏览器 Send to Kindle 云端传书”标签页仅列出 `PDF、DOC、DOCX、TXT、RTF、HTM、HTML、PNG、GIF、JPG、JPEG、BMP、EPUB`，并排除超过 200 MB 的文件。两个标签页顶部均提供按常见使用频率排列的格式快速筛选，标签后显示对应书籍数量，数量为 0 的格式会自动隐藏。两个标签页共用当前排序方式。页面采用全宽响应式布局，尽量利用 Kindle 屏幕空间，并沿用“方序语音”的深靛蓝、紫色渐变、圆角卡片与轻盈留白。品牌声纹由轻量 CSS 绘制，不加载外部字体或图片。

## 直接使用已编译版本

前往 [GitHub Releases](https://github.com/goldenwind/kindle-transfer/releases) 下载已编译版本；从源码自行构建时，产物默认放在本地 `dist` 目录：

| 文件 | 系统 | 支持架构 |
| --- | --- | --- |
| `kindle-send-windows-amd64.exe` | Windows | x86-64（绝大多数 Windows 电脑） |
| `Fangxu-Kindle-Transfer-macOS.zip` | macOS 双击版 | Intel、Apple Silicon（M1/M2/M3/M4 等） |
| `kindle-send-macos-universal` | macOS 命令行版 | Intel、Apple Silicon |
| `kindle-send-linux-amd64` | Linux | x86-64 |
| `kindle-send-linux-arm64` | Linux | ARM64（树莓派、ARM 服务器等） |

## 双击启动

启动成功后，程序会自动在电脑默认浏览器中打开控制页。控制页提供：

- Kindle 应打开的局域网地址
- 当前共享目录和电子书列表
- 原生“选择文件夹”按钮
- 手动输入目录路径
- 独立的 Send to Kindle 格式标签页
- 两个标签页顶部的格式快速筛选
- 时间倒序与文件名排序切换
- 点击书籍条目复制文件绝对路径
- 每本书对应的“打开 Send to Kindle”按钮
- 停止传书服务按钮

程序默认只启动一个服务实例。服务运行期间再次双击 Windows 程序或 macOS App，会自动验证并打开之前启动的控制页；原服务的共享目录和端口保持不变。

目录设置、文件路径复制及整个“电脑浏览器 Send to Kindle 云端传书”标签页只在电脑端控制页显示。Kindle 和其他局域网设备只显示“Kindle 内置浏览器下载”标签页及可直接下载的电子书。

### Windows

1. 双击 `kindle-send-windows-amd64.exe`。
2. Windows 防火墙询问时，允许程序访问“专用网络”。
3. 在自动打开的电脑端页面中点击“选择文件夹...”，选择电子书目录。
4. 在 Kindle 浏览器中输入页面显示的地址。

命令行启动方式：

   ```powershell
   .\kindle-send-windows-amd64.exe
   ```

还可以使用 `--dir` 直接指定电子书目录：

```powershell
.\kindle-send-windows-amd64.exe --dir "D:\电子书"
```

### macOS

1. 解压 `Fangxu-Kindle-Transfer-macOS.zip`。
2. 将 `Kindle传书.app` 移到任意位置并双击打开。
3. 在自动打开的页面中点击“选择文件夹...”，选择电子书目录。
4. 在 Kindle 浏览器中输入页面显示的地址。

如果 macOS 阻止首次打开，请右键点击 `Kindle传书.app`，选择“打开”，再确认一次。

需要使用命令行时，可以运行 `kindle-send-macos-universal`：

```bash
chmod +x kindle-send-macos-universal
./kindle-send-macos-universal
```

使用 `--dir` 直接指定电子书目录：

```bash
./kindle-send-macos-universal --dir "/Users/你的用户名/Books"
```

### Linux

根据设备架构下载 `kindle-send-linux-amd64` 或 `kindle-send-linux-arm64`，然后执行：

```bash
chmod +x kindle-send-linux-amd64
./kindle-send-linux-amd64
```

ARM64 设备请将上述文件名替换为 `kindle-send-linux-arm64`。程序会通过 `xdg-open` 自动打开控制页；桌面环境安装了 `zenity` 或 `kdialog` 时，可以直接使用“选择文件夹”按钮，否则可在页面中手动输入共享目录。

指定电子书目录或固定端口：

```bash
./kindle-send-linux-amd64 --dir "/home/yourname/Books" --listen 0.0.0.0:9000
```

## 从源码运行

```bash
go run . --dir "/你的/电子书目录"
```

无论使用编译版本还是源码运行，启动后终端都会显示类似地址：

```text
电子书根目录: /你的/电子书目录
Kindle 访问: http://192.168.1.10:49152
```

确保电脑和 Kindle 在同一局域网，然后在 Kindle 浏览器中打开 `Kindle 访问`后面的地址。

目录可以使用绝对路径或相对路径。包含空格或中文时，请用引号包住路径。

默认监听 `0.0.0.0:0`，其中端口 `0` 表示由系统自动选择当前空闲端口。每次完整停止并重新启动后，端口可能不同，请以控制页或终端输出为准。

如果需要固定端口，也可以同时设置目录和监听端口，例如：

```bash
./kindle-send-macos-universal --dir "/Users/me/Books" --listen 0.0.0.0:9000
```

Windows 对应命令：

```powershell
.\kindle-send-windows-amd64.exe --dir "D:\Books" --listen 0.0.0.0:9000
```

未指定 `--dir` 时，服务根目录是当前用户的“下载/Downloads”目录；如果该目录不存在，才会回退到执行命令时所在的当前目录。

如果不希望启动后自动打开电脑浏览器，可以添加 `--no-open`：

```bash
./kindle-send-macos-universal --no-open
```

## 重新编译

需要 Go 1.22 或更高版本。在项目目录执行：

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

# 合并为 macOS Universal Binary（需要在 macOS 上执行）
lipo -create -output dist/kindle-send-macos-universal \
  /tmp/kindle-send-darwin-amd64 /tmp/kindle-send-darwin-arm64
chmod +x dist/kindle-send-macos-universal
```

macOS 双击版的应用壳文件位于 `packaging/macos`，发布包由项目中的 Universal Binary、`Info.plist` 和启动脚本组成。

## 注意

- 服务没有登录验证，只建议在可信的家庭局域网中临时使用。
- 使用期间请保持程序运行；可在电脑端控制页点击“停止传书服务”，命令行下也可以按 `Ctrl+C`。
- Kindle 型号和系统版本不同，实际可直接打开的格式可能不同。
- Send to Kindle 需要联网并登录与 Kindle 关联的亚马逊账号，文件会通过亚马逊服务传输。
- 不支持浏览器下载的格式不会通过伪造 `.mobi` 扩展名下载；伪装只会让浏览器放行，并不能转换文件格式。

## 开源许可

方序传书使用 [MIT License](./LICENSE) 开源。欢迎提交 Issue、功能建议和 Pull Request。

如果这个免费项目为你带来了便利，欢迎在应用首页扫码支持，也欢迎关注微信公众号「一灯 AI」。
