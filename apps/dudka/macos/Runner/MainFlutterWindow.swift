import Cocoa
import FlutterMacOS

class MainFlutterWindow: NSWindow {
  override func awakeFromNib() {
    let flutterViewController = FlutterViewController()
    let windowFrame = self.frame
    self.contentViewController = flutterViewController
    self.setFrame(windowFrame, display: true)

    RegisterGeneratedPlugins(registry: flutterViewController)

    let desktopChannel = FlutterMethodChannel(
      name: "team.zamoo.dudka/desktop",
      binaryMessenger: flutterViewController.engine.binaryMessenger
    )
    desktopChannel.setMethodCallHandler { call, result in
      guard call.method == "setBadge", let count = call.arguments as? Int else {
        result(FlutterMethodNotImplemented)
        return
      }
      NSApp.dockTile.badgeLabel = count > 0 ? String(min(count, 999)) : nil
      result(nil)
    }

    super.awakeFromNib()
  }
}
