# Scene bootstrap recovery

Clean-room analysis of the archived Survival Heroes 3.3.3 client is tracing the early scene-entry messages immediately following login/role selection:

- `SERVER_PROPERTY_TABLE`
- `SERVER_RECORD_TABLE`
- `SERVER_ENTRY_SCENE` / `SERVER_ENTRY_SCENE_EX`

Only recovered protocol facts are promoted into the compatibility server; proprietary client assets are not committed.
