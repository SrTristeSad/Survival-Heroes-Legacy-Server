# Scene composite key proof

`SERVER_ENTRY_SCENE` formats two `uint32` values as `%u-%u` and looks the resulting text up in the client's scene registry. The archived `res\\share\\scenes.ini` defines both `Scene` (scene ID) and `Member` (scene-server number).

## Proven order

The `SceneBaseModule::LoadSceneBaseInfo` path opens `share/scenes.ini`, enumerates each numeric section and converts the section name to an integer. It then reads the `Member` value for that same section.

Immediately before the scene-add virtual call, the released ARM64 client places:

```text
w1 = numeric INI section / Scene ID
w2 = Member value
x3 = Script
x4 = Config
```

That scene-add path therefore receives `Scene` first and `Member` second. Elsewhere the client represents the same two-component scene identity as a 64-bit pair whose low 32 bits are formatted first and high 32 bits second with `%u-%u`.

Combined with `GameReceiver::ServerEntryScene`, which reads its two scene components in packet order and formats them with that same `%u-%u` representation, the runtime lookup key is:

```text
Scene-Member
```

not `Member-Scene`.

## First controlled scene

The archived registry contains the original MOBA scene:

```text
section/Scene = 119
Member = 0
Config = ini\\scene\\moba_chiji
Clonable = 1
BattleWarID = 1
PreLoad = 1
```

Therefore the client-known, preloaded base key for that map is:

```text
119-0
```

The compatibility server uses `119` / `0` only as the default **opt-in experiment** for `SERVER_ENTRY_SCENE`; emission remains disabled unless `MEMBER_SEND_ENTRY_SCENE=true`. This keeps static-analysis facts separate from claims about runtime success: reaching the lobby/map still requires a real patched-client test.
