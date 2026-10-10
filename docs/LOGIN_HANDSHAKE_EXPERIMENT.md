# Survival Heroes 3.3.3 — experimento de handshake membro/login

Este documento descreve apenas comportamentos recuperados do cliente Android 3.3.3 arquivado, além de hipóteses de teste explicitamente identificadas.

## Sequência confirmada no fio

A conexão membro/login usa `LenSplitProtocol`:

```text
uint32_le payload_length
byte      payload[payload_length]
```

Opcodes recuperados do cliente:

```text
0x00 CLIENT_GET_VERIFY
0x01 CLIENT_RET_ENCODE
0x02 CLIENT_LOGIN
0x03 CLIENT_WORLD_INFO
0x04 CLIENT_CHOOSE_ROLE
0x05 CLIENT_CREATE_ROLE
```

Opcodes recuperados do servidor:

```text
0x01 SERVER_SET_VERIFY
0x02 SERVER_SET_ENCODE
0x04 SERVER_LOGIN_SUCCEED
0x05 SERVER_WORLD_INFO
0x09 SERVER_PROPERTY_TABLE
0x0A SERVER_RECORD_TABLE
0x0B SERVER_ENTRY_SCENE
```

## `SERVER_SET_VERIFY`

`GameReceiver::ServerSetVerify` exige pelo menos 13 bytes de payload.

```text
offset 0   uint8     opcode = 0x01
offset 1   uint32_le field0
offset 5   uint32_le mode
offset 9   byte[4]   padding quando mode != 1
```

Quando `mode == 1`, o receiver exige pelo menos 25 bytes e copia exatamente 16 bytes a partir do offset 9:

```text
offset 9   byte[16] key
```

O significado de alto nível de `field0` e `mode` ainda está em validação e continua sem nome semântico no código.

## `SERVER_SET_ENCODE`

O receiver ARM64 da versão 3.3.3 verifica apenas que o payload possui mais de 61 bytes. Na rotina observada, o corpo do pacote não é lido antes do callback `on_set_encode`.

O servidor de compatibilidade possui um builder experimental para a forma mínima aceita:

```text
offset 0   uint8     opcode = 0x02
offset 1   byte[61]  payload de pesquisa preenchido com zero
```

São **62 bytes de payload**, além do prefixo normal de quatro bytes do `LenSplitProtocol`.

## Chaves de teste controlado

As respostas experimentais permanecem desligadas por padrão.

```env
MEMBER_SEND_SET_VERIFY=true
MEMBER_SET_VERIFY_FIELD0=0
MEMBER_SET_VERIFY_MODE=0
MEMBER_SET_VERIFY_KEY_HEX=
MEMBER_SEND_SET_ENCODE=true
MEMBER_SEND_LOGIN_SUCCESS=true
MEMBER_SEND_WORLD_INFO=true
MEMBER_SEND_SCENE_TABLES=true
MEMBER_SEND_ENTRY_SCENE=true
MEMBER_ENTRY_SCENE_A=119
MEMBER_ENTRY_SCENE_B=0
```

Para `MEMBER_SET_VERIFY_MODE=1`, `MEMBER_SET_VERIFY_KEY_HEX` deve conter exatamente 16 bytes / 32 caracteres hexadecimais.

O atalho recomendado é:

```bash
sh scripts/enable-legacy-flow.sh
```

## Experimento com cliente do próprio testador

Com um cliente 3.3.3 repacotado para apontar para a VPS e a lista de servidores apontando para TCP/7001, acompanhe:

```bash
docker compose logs -f query member
```

Fluxo esperado atualmente:

```text
CLIENT_GET_VERIFY
SERVER_SET_VERIFY
CLIENT_RET_ENCODE
SERVER_SET_ENCODE
CLIENT_LOGIN
SERVER_LOGIN_SUCCEED
CLIENT_WORLD_INFO
SERVER_WORLD_INFO
CLIENT_CHOOSE_ROLE
SERVER_PROPERTY_TABLE
SERVER_RECORD_TABLE
SERVER_ENTRY_SCENE 119-0
```

O servidor registra somente metadados estruturais de `CLIENT_LOGIN`; ele não imprime conta, senha, token ou conteúdo do identificador de dispositivo.

## Limite atual

O framing de login, `SERVER_LOGIN_SUCCEED`, a gramática de registros de personagem, criação/seleção de personagem, `WORLD_INFO`, tabelas iniciais e `ENTRY_SCENE` já foram recuperados.

O limite atual deixou de ser a análise estática do handshake: o próximo passo é validar em execução real até onde o APK avança e observar o primeiro pacote/estado que ainda impede a entrada completa no lobby ou mapa.
