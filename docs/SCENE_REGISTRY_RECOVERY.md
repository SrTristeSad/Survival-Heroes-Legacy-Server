# Recuperação do registro de cenas

A análise do OBB do cliente 3.3.3 identificou o registro principal de cenas em:

```text
res\share\scenes.ini
```

Também foram localizados arquivos relacionados, incluindo `scene_group.xml` e configurações de mapas como `moba_chiji.xml`.

## Campos relevantes

Cada seção numérica do `scenes.ini` representa um `Scene ID`. Entre os campos associados aparecem:

- `Member`
- `Config`
- `Script`
- `Clonable`
- `BattleWarID`
- `PreLoad`

A análise nativa confirmou que a identidade composta usada em runtime é `Scene-Member`.

## Cena MOBA confirmada

```text
Scene = 119
Member = 0
Config = ini\scene\moba_chiji
Clonable = 1
BattleWarID = 1
PreLoad = 1
```

Chave resultante:

```text
119-0
```

Os arquivos proprietários extraídos durante a pesquisa não são versionados neste repositório; somente fatos necessários para interoperabilidade são documentados.
