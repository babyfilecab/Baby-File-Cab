#define MyAppName "BabyFileCab"
#define MyAppVersion "1.0.0"
#define MyAppPublisher "BabyFileCab"
#define MyAppExeName "BabyFileCab.exe"

[Setup]
AppId={{c5d99b03-876f-4ec7-a420-cfb988950aa5}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}

DefaultDirName={localappdata}\Programs\BabyFileCab
DefaultGroupName=BabyFileCab

PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible

OutputDir=output
OutputBaseFilename=BabyFileCab-Setup-v1.0.0

SetupIconFile=..\assets\BabyFileCab.ico
UninstallDisplayIcon={app}\BabyFileCab.exe

Compression=lzma2
SolidCompression=yes
WizardStyle=modern

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Additional shortcuts:"

[Files]
Source: "..\dist\BabyFileCab\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{autoprograms}\BabyFileCab"; Filename: "{app}\BabyFileCab.exe"
Name: "{autodesktop}\BabyFileCab"; Filename: "{app}\BabyFileCab.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\BabyFileCab.exe"; Description: "Launch BabyFileCab"; Flags: nowait postinstall skipifsilent
