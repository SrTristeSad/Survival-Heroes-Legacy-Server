# Recuperação do bootstrap de cena

A análise independente do cliente Survival Heroes 3.3.3 recuperou as primeiras mensagens usadas logo após login e seleção de personagem.

## Mensagens confirmadas

```text
0x09 SERVER_PROPERTY_TABLE
0x0A SERVER_RECORD_TABLE
0x0B SERVER_ENTRY_SCENE
```

## `SERVER_PROPERTY_TABLE`

O cliente aceita uma forma mínima válida contendo apenas opcode e contagem zero:

```text
uint8  opcode = 0x09
uint16 count  = 0
```

Isso permite inicializar a tabela de propriedades sem inventar dados ainda não conhecidos.

## `SERVER_RECORD_TABLE`

Também aceita uma forma mínima válida:

```text
uint8  opcode = 0x0A
uint16 count  = 0
```

## `SERVER_ENTRY_SCENE`

O pacote mínimo recuperado possui 15 bytes e inclui dois componentes `uint32` usados para identificar a cena.

O cliente formata esses dois valores como:

```text
%u-%u
```

A análise do loader de `scenes.ini` confirmou que a ordem correta é:

```text
Scene-Member
```

## Cena inicial de teste

No registro original do cliente foi encontrada a cena:

```text
Scene = 119
Member = 0
Config = ini\scene\moba_chiji
Clonable = 1
BattleWarID = 1
PreLoad = 1
```

Portanto a chave conhecida pelo cliente é:

```text
119-0
```

## Implementação atual

Quando habilitado, o `member-server` pode responder a `CLIENT_CHOOSE_ROLE` com:

```text
SERVER_PROPERTY_TABLE vazio
→ SERVER_RECORD_TABLE vazio
→ SERVER_ENTRY_SCENE 119-0
```

Esse comportamento é experimental e permanece desligado por padrão. Para ativar todo o fluxo recuperado:

```bash
sh scripts/enable-legacy-flow.sh
```

A análise estática prova a estrutura e a chave da cena, mas a entrada real no lobby/mapa só será considerada confirmada depois do teste com um APK 3.3.3 apontado para a VPS.
