; Packs dist/app into the installer. installer/build.ps1 fills that folder first.

#ifndef AppVersion
  #define AppVersion "0.0.0"
#endif

[Setup]
AppId={{6F1D2C54-8B7E-4C3A-9E21-5D0A7B3F4E18}
AppName=TotalKrieg
AppVersion={#AppVersion}
AppPublisher=xetra11
DefaultDirName={autopf}\TotalKrieg
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
OutputDir=..\dist
OutputBaseFilename=TotalKrieg-setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
UninstallDisplayIcon={app}\TotalKrieg.exe

[Tasks]
Name: desktopicon; Description: "Create a desktop shortcut"; Flags: unchecked

; An update may bring a MiusMissionMaker with fewer files.
[InstallDelete]
Type: filesandordirs; Name: "{app}\MiusMissionMaker"

[Files]
Source: "..\dist\app\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs

; generated missions
[UninstallDelete]
Type: filesandordirs; Name: "{localappdata}\TotalKrieg"

[Icons]
Name: "{autoprograms}\TotalKrieg"; Filename: "{app}\TotalKrieg.exe"
Name: "{autodesktop}\TotalKrieg"; Filename: "{app}\TotalKrieg.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\TotalKrieg.exe"; Description: "Start TotalKrieg"; Flags: nowait postinstall skipifsilent
