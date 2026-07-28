#ifndef AppVersion
  #define AppVersion "0.0.0"
#endif
#ifndef BundleDir
  #error BundleDir is required
#endif
#ifndef OutputDir
  #define OutputDir "."
#endif

[Setup]
AppId={{B85277E5-578B-4FF0-A2C6-CC9CFA042142}
AppName=ДУДКА
AppVersion={#AppVersion}
AppPublisher=Студия ПО «Замутим»
DefaultDirName={localappdata}\Programs\Dudka
DefaultGroupName=ДУДКА
DisableProgramGroupPage=yes
OutputDir={#OutputDir}
OutputBaseFilename=dudka-windows-amd64-setup
Compression=lzma2
SolidCompression=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
SetupIconFile={#BundleDir}\data\flutter_assets\windows\runner\resources\app_icon.ico
UninstallDisplayIcon={app}\dudka.exe
WizardStyle=modern

[Tasks]
Name: desktopicon; Description: "Создать значок на рабочем столе"; GroupDescription: "Дополнительно:"; Flags: unchecked

[Files]
Source: "{#BundleDir}\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{group}\ДУДКА"; Filename: "{app}\dudka.exe"; WorkingDir: "{app}"
Name: "{autodesktop}\ДУДКА"; Filename: "{app}\dudka.exe"; WorkingDir: "{app}"; Tasks: desktopicon

[Run]
Filename: "{app}\dudka.exe"; Description: "Запустить ДУДКУ"; Flags: nowait postinstall skipifsilent
