; Carnival Constructor Classic (Demo) - standalone setup. Build: bash build.sh, then ISCC.exe /DGameVersion=<version> installer\CarnivalSetup.iss
; (c) 2026 Ironite
#define GameName    "Carnival Constructor Classic"
#define Bundle      "CarnivalConstructor"
#define Exe         "CarnivalConstructor.exe"
#ifndef GameVersion
  #define GameVersion "0.1.0"
#endif
#define AppGuid     "{{8C3E5B21-7A4D-4F0E-9B62-3D1F6A8E2C47}"
#define Art         "carnival"
#define Tagline     "A traveling carnival sim (free demo). Build rides, midway games and food stands, run the busy nights, and keep the town coming back."
#define SaveWhat    "Carnival Constructor Classic settings and records"
#include "GameSetup.iss"

[InstallDelete]
; shortcuts from before the game was renamed "Carnival Constructor Classic"
Type: files; Name: "{group}\Carnival Constructor.lnk"
Type: files; Name: "{autodesktop}\Carnival Constructor.lnk"
