# Scene registry recovery

The archived 3.3.3 client resolves `SERVER_ENTRY_SCENE` using a textual key formatted as `%u-%u` from the two little-endian `uint32` values at packet offsets `+5` and `+9`.

This one-shot analysis extracts only the small scene-registry/configuration files needed to identify valid client scene keys. Proprietary assets remain outside the repository.
