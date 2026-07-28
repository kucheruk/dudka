#include "flutter_window.h"

#include <algorithm>
#include <iomanip>
#include <optional>
#include <sstream>

#include "flutter/generated_plugin_registrant.h"

namespace {

HICON CreateBadgeIcon(int count) {
  constexpr int kSize = 32;
  HDC screen = GetDC(nullptr);
  HDC memory = CreateCompatibleDC(screen);
  HBITMAP color = CreateCompatibleBitmap(screen, kSize, kSize);
  HBITMAP previous = static_cast<HBITMAP>(SelectObject(memory, color));

  RECT rect{0, 0, kSize, kSize};
  HBRUSH transparent = CreateSolidBrush(RGB(0, 0, 0));
  FillRect(memory, &rect, transparent);
  DeleteObject(transparent);

  HBRUSH red = CreateSolidBrush(RGB(255, 59, 48));
  HGDIOBJ old_brush = SelectObject(memory, red);
  HGDIOBJ old_pen = SelectObject(memory, GetStockObject(NULL_PEN));
  Ellipse(memory, 1, 1, kSize - 1, kSize - 1);

  SetBkMode(memory, TRANSPARENT);
  SetTextColor(memory, RGB(255, 255, 255));
  const int font_size = count > 99 ? 13 : 17;
  HFONT font = CreateFontW(
      -font_size, 0, 0, 0, FW_BOLD, FALSE, FALSE, FALSE, DEFAULT_CHARSET,
      OUT_DEFAULT_PRECIS, CLIP_DEFAULT_PRECIS, CLEARTYPE_QUALITY,
      DEFAULT_PITCH | FF_SWISS, L"Segoe UI");
  HGDIOBJ old_font = SelectObject(memory, font);
  const std::wstring text = count > 999 ? L"999+" : std::to_wstring(count);
  DrawTextW(memory, text.c_str(), -1, &rect,
            DT_CENTER | DT_VCENTER | DT_SINGLELINE);

  SelectObject(memory, old_font);
  SelectObject(memory, old_pen);
  SelectObject(memory, old_brush);
  SelectObject(memory, previous);
  DeleteObject(font);
  DeleteObject(red);
  DeleteDC(memory);
  ReleaseDC(nullptr, screen);

  HBITMAP mask = CreateBitmap(kSize, kSize, 1, 1, nullptr);
  ICONINFO info{};
  info.fIcon = TRUE;
  info.hbmColor = color;
  info.hbmMask = mask;
  HICON icon = CreateIconIndirect(&info);
  DeleteObject(mask);
  DeleteObject(color);
  return icon;
}

}  // namespace

FlutterWindow::FlutterWindow(const flutter::DartProject& project)
    : project_(project) {}

FlutterWindow::~FlutterWindow() {
  if (badge_icon_) {
    DestroyIcon(badge_icon_);
  }
  if (taskbar_) {
    taskbar_->Release();
  }
}

bool FlutterWindow::OnCreate() {
  if (!Win32Window::OnCreate()) {
    return false;
  }

  RECT frame = GetClientArea();

  // The size here must match the window dimensions to avoid unnecessary surface
  // creation / destruction in the startup path.
  flutter_controller_ = std::make_unique<flutter::FlutterViewController>(
      frame.right - frame.left, frame.bottom - frame.top, project_);
  // Ensure that basic setup of the controller was successful.
  if (!flutter_controller_->engine() || !flutter_controller_->view()) {
    return false;
  }
  RegisterPlugins(flutter_controller_->engine());
  desktop_channel_ =
      std::make_unique<flutter::MethodChannel<flutter::EncodableValue>>(
          flutter_controller_->engine()->messenger(),
          "team.zamoo.dudka/desktop",
          &flutter::StandardMethodCodec::GetInstance());
  desktop_channel_->SetMethodCallHandler(
      [this](const auto& call, auto result) {
        if (call.method_name() != "setBadge") {
          result->NotImplemented();
          return;
        }
        int count = 0;
        const auto* arguments = call.arguments();
        if (arguments) {
          if (const auto* value = std::get_if<int32_t>(arguments)) {
            count = *value;
          } else if (const auto* value = std::get_if<int64_t>(arguments)) {
            count = static_cast<int>(*value);
          }
        }
        SetBadge(std::max(0, count));
        result->Success();
      });

  CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED);
  if (SUCCEEDED(CoCreateInstance(CLSID_TaskbarList, nullptr,
                                 CLSCTX_INPROC_SERVER, IID_PPV_ARGS(&taskbar_)))) {
    taskbar_->HrInit();
  }
  SetChildContent(flutter_controller_->view()->GetNativeWindow());

  flutter_controller_->engine()->SetNextFrameCallback([&]() {
    this->Show();
  });

  // Flutter can complete the first frame before the "show window" callback is
  // registered. The following call ensures a frame is pending to ensure the
  // window is shown. It is a no-op if the first frame hasn't completed yet.
  flutter_controller_->ForceRedraw();

  return true;
}

void FlutterWindow::OnDestroy() {
  desktop_channel_.reset();
  if (flutter_controller_) {
    flutter_controller_ = nullptr;
  }

  Win32Window::OnDestroy();
}

void FlutterWindow::SetBadge(int count) {
  if (!taskbar_) {
    return;
  }
  if (badge_icon_) {
    DestroyIcon(badge_icon_);
    badge_icon_ = nullptr;
  }
  if (count > 0) {
    badge_icon_ = CreateBadgeIcon(count);
  }
  taskbar_->SetOverlayIcon(GetHandle(), badge_icon_,
                           count > 0 ? L"Непрочитанные сообщения" : L"");
}

LRESULT
FlutterWindow::MessageHandler(HWND hwnd, UINT const message,
                              WPARAM const wparam,
                              LPARAM const lparam) noexcept {
  // Give Flutter, including plugins, an opportunity to handle window messages.
  if (flutter_controller_) {
    std::optional<LRESULT> result =
        flutter_controller_->HandleTopLevelWindowProc(hwnd, message, wparam,
                                                      lparam);
    if (result) {
      return *result;
    }
  }

  switch (message) {
    case WM_FONTCHANGE:
      flutter_controller_->engine()->ReloadSystemFonts();
      break;
  }

  return Win32Window::MessageHandler(hwnd, message, wparam, lparam);
}
