# Scene composite key proof

`SERVER_ENTRY_SCENE` formats two `uint32` values as `%u-%u` and looks the resulting text up in the client's scene registry. The archived `res\\share\\scenes.ini` defines both `Scene` (scene ID) and `Member` (scene-server number).

A focused one-shot disassembly traces the `scenes.ini` loader to prove whether the runtime key is `Scene-Member`, `Member-Scene`, or another pairing before the compatibility server emits `ENTRY_SCENE`.
