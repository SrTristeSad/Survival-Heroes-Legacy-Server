# Protocolo membro/login do Survival Heroes 3.3.3

Este documento contém apenas detalhes confirmados a partir do cliente Android 3.3.3 arquivado. Campos ainda desconhecidos permanecem sem nome semântico em vez de receber rótulos por adivinhação.

## Framing TCP externo

O caminho antigo de membro/jogo usa `LenSplitProtocol`.

`LenSplitProtocol::m_nProtocolSize` vale `4`, e a rotina de recepção interpreta os quatro primeiros bytes como o tamanho exato do payload:

```text
uint32_le payload_length
byte      payload[payload_length]
```

O primeiro byte de cada payload decodificado é o opcode antigo de `GameReceiver` / `GameSender`.

Esse framing está implementado em `internal/legacylogin.FrameWirePayload` e `ParseWireFrame`.

## Opcodes confirmados cliente → servidor

| Opcode | Mensagem |
|---:|---|
| `0x00` | `CLIENT_GET_VERIFY` |
| `0x01` | `CLIENT_RET_ENCODE` |
| `0x02` | `CLIENT_LOGIN` |
| `0x03` | `CLIENT_WORLD_INFO` |
| `0x04` | `CLIENT_CHOOSE_ROLE` |
| `0x05` | `CLIENT_CREATE_ROLE` |
| `0x06` | `CLIENT_DELETE_ROLE` |

`CLIENT_GET_VERIFY` possui payload de um único byte. Com o framing externo completo:

```text
01 00 00 00  00
^^^^^^^^^^^  ^^
payload_len  CLIENT_GET_VERIFY
```

## Opcodes confirmados servidor → cliente

A tabela de dispatch ARM64 de `GameReceiver` foi decodificada diretamente.

| Opcode | Handler/mensagem |
|---:|---|
| `0x01` | `SERVER_SET_VERIFY` |
| `0x02` | `SERVER_SET_ENCODE` |
| `0x03` | `SERVER_ERROR_CODE` |
| `0x04` | `SERVER_LOGIN_SUCCEED` |
| `0x05` | `SERVER_WORLD_INFO` |
| `0x09` | `SERVER_PROPERTY_TABLE` |
| `0x0A` | `SERVER_RECORD_TABLE` |
| `0x0B` | `SERVER_ENTRY_SCENE` |

## `SERVER_SET_VERIFY` (`0x01`)

O receiver exige no mínimo 13 bytes:

```text
offset  size  tipo       significado
0       1     uint8      opcode = 0x01
1       4     uint32_le  field0 (significado ainda não confirmado)
5       4     uint32_le  verify_mode
9       ...              dados dependentes do modo
```

Se `verify_mode == 1`, o cliente exige pelo menos 25 bytes e consome uma chave de 16 bytes iniciando em offset 9.

Para os demais modos, o builder atual mantém quatro bytes zero nos offsets 9..12 porque o receiver original ainda exige o mínimo de 13 bytes.

Implementação: `legacylogin.BuildServerSetVerify`.

## `SERVER_SET_ENCODE` (`0x02`)

O receiver exige payload maior que `0x3d`, portanto no mínimo 62 bytes, antes de emitir `on_set_encode`.

Na rotina observada, os bytes após o opcode não são consumidos antes do callback. O servidor mantém um builder experimental de 62 bytes para testes controlados.

## `SERVER_LOGIN_SUCCEED` (`0x04`)

O cliente exige no mínimo `0x29` (41) bytes.

Prefixo fixo:

```text
offset  size  tipo
0       1     uint8      opcode = 0x04
1       4     uint32_le  value0
5       4     uint32_le  value1
9       4     uint32_le  value2
13      4     uint32_le  value3
17      4     uint32_le  value4
21      4     uint32_le  value5
25      4     uint32_le  value6
29      4     uint32_le  value7
33      4     uint32_le  value8
37      4     uint32_le  role_count
```

Os registros de personagem começam no offset 41 quando `role_count > 0`.

Um `role_count=0` é válido e é usado no experimento atual para alcançar o caminho original de criação de personagem sem inventar dados proprietários.

## Registros de personagem

Cada personagem começa com:

```text
uint16 parameter_count
```

Depois vêm parâmetros tipados. Tags confirmadas:

| Tag | Valor no fio |
|---:|---|
| `2` | `uint32` |
| `3` | `uint64` |
| `4` | `uint32` |
| `5` | `uint64` |
| `6` | `uint32 byte_length` + string de bytes terminada em NUL |
| `7` | `uint32 byte_length` + UTF-16LE terminado em `0x0000` |
| `8` | dois `uint32` consecutivos |

O encoder fica em `internal/legacylogin/role_record.go`.

## `CLIENT_CREATE_ROLE` (`0x05`)

Prefixo mínimo recuperado de 73 bytes:

```text
offset 0     uint8   opcode = 0x05
offset 1     uint32  field0
offset 5     uint16  name_utf16le[33]
offset 71    uint16  extra_parameter_count
offset 73    ...     parametros opcionais
```

## `CLIENT_CHOOSE_ROLE` (`0x04`)

A requisição de escolha de personagem foi recuperada e já é validada pelo `member-server`. O identificador do personagem ocupa uma área UTF-16LE fixa e é seguido por `scene_info` como string C.

## `WORLD_INFO`

`CLIENT_WORLD_INFO` usa opcode `0x03`.

`SERVER_WORLD_INFO` usa opcode `0x05` e a forma:

```text
uint8  opcode
uint16 field0
uint16 texto_utf16le[] + NUL
```

## Bootstrap de cena

`SERVER_PROPERTY_TABLE` (`0x09`) e `SERVER_RECORD_TABLE` (`0x0A`) aceitam uma forma vazia válida:

```text
uint8  opcode
uint16 count = 0
```

`SERVER_ENTRY_SCENE` (`0x0B`) foi recuperado e o cliente forma a identidade de cena usando dois `uint32` em ordem `Scene-Member`.

A cena MOBA original confirmada no registro do cliente é:

```text
Scene = 119
Member = 0
chave = 119-0
```

## Fluxo experimental atual

```text
CLIENT_GET_VERIFY
→ SERVER_SET_VERIFY

CLIENT_RET_ENCODE
→ SERVER_SET_ENCODE

CLIENT_LOGIN
→ SERVER_LOGIN_SUCCEED

CLIENT_WORLD_INFO
→ SERVER_WORLD_INFO

CLIENT_CHOOSE_ROLE
→ SERVER_PROPERTY_TABLE
→ SERVER_RECORD_TABLE
→ SERVER_ENTRY_SCENE 119-0
```

A emissão continua opt-in e pode ser ativada por:

```bash
sh scripts/enable-legacy-flow.sh
```

## Privacidade no servidor de compatibilidade

`cmd/member-server` valida os pacotes reais do protocolo sem registrar conteúdo sensível de conta, senha, token ou identificador do dispositivo. Apenas tamanhos, seletores não secretos e metadados estruturais necessários para a pesquisa são registrados.
