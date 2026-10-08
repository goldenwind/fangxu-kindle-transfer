// Run on macOS: swift scripts/build-dmg-background.swift
// Commit the generated PNG so all macOS release builds use the same artwork.
import AppKit

let root = URL(fileURLWithPath: #filePath).deletingLastPathComponent().deletingLastPathComponent()
let config = try JSONSerialization.jsonObject(with: Data(contentsOf: root.appendingPathComponent("src-tauri/tauri.conf.json"))) as! [String: Any]
let dmg = ((config["bundle"] as! [String: Any])["macOS"] as! [String: Any])["dmg"] as! [String: Any]
let window = dmg["windowSize"] as! [String: Int]
// Finder's window bounds include its title bar. Keep the background and all
// artwork comfortably inside the content area, even with legacy scrollbars.
let width = window["width"]!
let height = window["height"]! - 50
let image = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: width, pixelsHigh: height,
    bitsPerSample: 8, samplesPerPixel: 4, hasAlpha: true, isPlanar: false,
    colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0)!
NSGraphicsContext.saveGraphicsState()
NSGraphicsContext.current = NSGraphicsContext(bitmapImageRep: image)

func color(_ r: CGFloat, _ g: CGFloat, _ b: CGFloat) -> NSColor {
    NSColor(srgbRed: r / 255, green: g / 255, blue: b / 255, alpha: 1)
}
let green = color(39, 77, 53)
color(247, 249, 242).setFill()
NSRect(x: 0, y: 0, width: width, height: height).fill()

func centered(_ text: String, top: CGFloat, size: CGFloat, weight: NSFont.Weight, ink: NSColor) {
    let style = NSMutableParagraphStyle()
    style.alignment = .center
    let attributes: [NSAttributedString.Key: Any] = [
        .font: NSFont.systemFont(ofSize: size, weight: weight),
        .foregroundColor: ink, .paragraphStyle: style,
    ]
    (text as NSString).draw(in: NSRect(x: 24, y: CGFloat(height) - top - 36,
        width: CGFloat(width) - 48, height: 36), withAttributes: attributes)
}

centered("拖动到 Applications 即可安装", top: 25, size: 25, weight: .semibold, ink: green)
centered("Drag the app into Applications to install", top: 65, size: 15,
    weight: .regular, ink: color(89, 104, 92))

let app = dmg["appPosition"] as! [String: Int]
let applications = dmg["applicationFolderPosition"] as! [String: Int]
let arrowY = CGFloat(height - app["y"]!)
let arrowStart = CGFloat(app["x"]! + 112)
let arrowEnd = CGFloat(applications["x"]! - 112)
let arrow = NSBezierPath()
arrow.lineWidth = 3
arrow.lineCapStyle = .round
arrow.lineJoinStyle = .round
arrow.move(to: NSPoint(x: arrowStart, y: arrowY))
arrow.line(to: NSPoint(x: arrowEnd, y: arrowY))
arrow.move(to: NSPoint(x: arrowEnd - 12, y: arrowY + 10))
arrow.line(to: NSPoint(x: arrowEnd, y: arrowY))
arrow.line(to: NSPoint(x: arrowEnd - 12, y: arrowY - 10))
green.setStroke()
arrow.stroke()

centered("安装后，从「应用程序」打开方序传书", top: 273, size: 13,
    weight: .regular, ink: color(111, 123, 111))
centered("Once installed, open Fangxu Kindle Transfer from Applications", top: 298, size: 12,
    weight: .regular, ink: color(111, 123, 111))

NSGraphicsContext.restoreGraphicsState()
// Background paths in the Tauri config are relative to src-tauri.
let destination = root.appendingPathComponent("src-tauri").appendingPathComponent(dmg["background"] as! String).standardizedFileURL
try image.representation(using: .png, properties: [:])!.write(to: destination)
print("Generated \(destination.path) (\(width) × \(height))")
