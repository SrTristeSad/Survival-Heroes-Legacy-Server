# Scene bootstrap recovery

Clean-room ARM64 analysis of the archived Survival Heroes 3.3.3 client has recovered the first three scene-bootstrap messages sufficiently to build parser-valid minimal packets without inventing gameplay data.

## Message opcodes

The original server message table is zero-based. Around the scene transition it contains:

```text
0x09 SERVER_PROPERTY_TABLE
0x0A SERVER_RECORD_TABLE
0x0B SERVER_ENTRY_SCENE
0x0C SERVER_EXIT_SCENE
```

## PROPERTY_TABLE

`GameReceiver::ServerPropertyTable` requires at least three bytes and reads a little-endian `uint16` count at offset `+1`. With count zero, no table entries are parsed and a packet of exactly three bytes is accepted:

```text
09 00 00
```

The receiver marks its property-table initialization flag after successful processing.

`internal/legacylogin.BuildEmptyServerPropertyTable()` implements this minimal form.

## RECORD_TABLE

`GameReceiver::ServerRecordTable` has the same parser-valid empty shape: opcode plus a little-endian zero `uint16` count:

```text
0A 00 00
```

The receiver independently marks its record-table initialization flag after successful processing.

`internal/legacylogin.BuildEmptyServerRecordTable()` implements this minimal form.

## ENTRY_SCENE fixed prefix

`GameReceiver::ServerEntryScene` requires at least 15 bytes. The recovered fixed prefix is:

```text
offset 0     uint8   opcode = 0x0B
offset 1     uint32  field0
offset 5     uint32  scene_a
offset 9     uint32  scene_b
offset 13    uint16  property_count
offset 15    ...     property payload
```

The released 3.3.3 handler does not consume `field0` in the recovered function path, so its business meaning remains intentionally unnamed.

The important part is the scene lookup: the client formats `scene_a` and `scene_b` using the literal format string `%u-%u`, then looks that key up in its already-loaded scene registry. If the pair is not present, the receiver enters its `(GameReceiver::ServerEntryScene)no scene` error path.

A zero-property prefix is parser-valid at the property-list layer, but it is **not** safe to emit with arbitrary scene numbers. `BuildServerEntrySceneZeroProperties` therefore exists as a wire-format builder only; the compatibility server must first recover a real key from the client scene registry.

Before processing `ENTRY_SCENE`, the receiver also ensures the PROPERTY_TABLE and RECORD_TABLE initialization paths have run. This is why the revival implementation treats those two empty-table packets as explicit prerequisites rather than skipping them.

## Current milestone

The remaining blocker for a controlled first scene-entry experiment is identifying at least one real `scene_a-scene_b` key from the archived client data. Small registry files such as `res\\share\\scenes.ini` and the scene XML metadata are being inspected for that mapping; proprietary graphical/game assets are not added to this repository.
