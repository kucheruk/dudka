# iOS build & install path (P084)

## What this machine can produce without Apple Developer signing

```bash
./scripts/build_ios_app.sh
# → dist/dudka-ios-Runner.app  (iphoneos, --no-codesign)
# → dist/dudka-ios-unsigned.zip
# → dist/BUILD-IOS.md (copy of install steps)
```

Unsigned `Runner.app` proves the Flutter iOS target compiles. It does **not** install on a
physical iPhone until signed.

## Path A — Ad-hoc / development (family device you own)

1. Apple ID with free or paid developer membership on a Mac with Xcode.
2. Open `apps/dudka/ios/Runner.xcworkspace` in Xcode.
3. Signing & Capabilities → Team = your Personal Team (or org).
4. Bundle ID: change `com.example.dudka` to something unique if needed.
5. Connect iPhone (USB or network), trust the Mac, enable Developer Mode on the phone.
6. Product → Run (or `flutter run -d <deviceId>`).
7. On the phone: Settings → General → VPN & Device Management → trust the developer cert.

Re-sign a CI artifact (optional):

```bash
codesign -f -s "Apple Development: Your Name (TEAMID)" \
  --entitlements path/to/entitlements.plist \
  dist/dudka-ios-Runner.app
ios-deploy --bundle dist/dudka-ios-Runner.app   # or Xcode Devices window
```

## Path B — TestFlight (wider family)

1. Paid Apple Developer Program.
2. Archive in Xcode (Product → Archive) or:

```bash
flutter build ipa --export-options-plist=ios/ExportOptions.plist
```

3. Upload to App Store Connect → TestFlight → add Internal/External testers.
4. Family installs via TestFlight app (still no Дудка cloud account — Apple ID only for store).

## Engine / sidecar

Same as Android: MVP GUI talks to loopback `dudkad`. Packaging a Go binary inside the iOS
sandbox needs a separate embed plan (not required to close P084 docs+unsigned device build).
For apartment LAN demo, run `dudkad` on a Mac/PC on the same Wi‑Fi until mobile embed lands.

## Acceptance for P084

| Check | Status |
| --- | --- |
| iOS Flutter scaffold in repo | yes (`apps/dudka/ios/`) |
| Device-arch release build script | `./scripts/build_ios_app.sh` |
| Ad-hoc + TestFlight path documented | this file |
| Physical install on this agent host | **blocked**: no codesign identity / no iPhone attached |
