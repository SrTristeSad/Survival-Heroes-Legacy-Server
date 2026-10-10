# Análise do cliente Survival Heroes 3.3.3

Este documento registra observações confirmadas por análise independente do cliente Android 3.3.3 arquivado. Fatos observados no cliente são mantidos separados de hipóteses que ainda precisam de validação em execução real.

## Cliente alvo

- Pacote: `com.snailgamesusa.survival`
- Versão: `3.3.3`
- Version code: `103`
- SHA-256 do XAPK arquivado: `b536ee1bc9a37ba8198aa5e6caeb6f008dce816d52d98c616e0477182462d98e`
- OBB principal: `main.103.com.snailgamesusa.survival.obb`

O projeto não redistribui esse cliente nem seus recursos proprietários.

## Runtime / engine

O cliente não é Unity. A análise estática mostra um runtime de jogo em C++ nativo com componentes relacionados ao Cocos2d-x, suporte a Lua e PhysX. Grande parte da rede está implementada em `libapp.so`.

Símbolos nativos relevantes incluem:

- `CNetWork::ConnectServer`
- `CNetWork::DisconnectServer`
- `CNetWork::SendData`
- `CNetWork::ConnectUdpServer`
- `CNetWork::SendUdpData`
- `CNetWork::StartUDPSession`
- `LenSplitProtocol::MakeSendData`
- `LenSplitProtocol::ProcessRecvData`

A pilha UDP expõe canais separados, incluindo confiável sequenciado, não confiável e não confiável sequenciado. O canal confiável sequenciado utiliza **KCP**.

**Consequência:** o socket TCP/JSON criado na fase inicial do projeto é apenas um laboratório de desenvolvimento. Ele não representa o transporte final de batalha do Survival Heroes.

## Configuração original de bootstrap

`assets/config.ini` contém:

```ini
[server_query]
Addr=164.52.38.62
Port=88
Issuer=10000
GameID=125
```

O IP histórico não deve ser consultado. Em testes de interoperabilidade, o endereço deve ser redirecionado para um servidor controlado pelo próprio testador.

Outras chaves úteis encontradas no cliente:

- `server_id`
- `server_port`
- `server_district`
- `server_issuer`
- `server_recommend`
- `server_page`
- `server_list.ini`

## Requisição de consulta de servidor na TCP/88

O cliente 3.3.3 conecta via TCP ao endereço de `[server_query]` e envia uma requisição fixa de 28 bytes.

`CMsgBuf::SetInt` grava diretamente valores nativos de 32 bits. No cliente Android little-endian analisado, a requisição corresponde a sete `uint32` consecutivos em little-endian:

| Índice | Valor | Significado |
|---:|---:|---|
| 0 | 24 | constante de framing/família |
| 1 | 1001 | comando de consulta |
| 2 | 1 | constante de versão/família |
| 3 | 1 | constante de versão/família |
| 4 | 22222 | magic/constante da família |
| 5 | 10000 | `Issuer` de `config.ini` |
| 6 | 125 | `GameID` de `config.ini` |

Bytes exatos para os valores padrão do cliente arquivado:

```text
18 00 00 00
E9 03 00 00
01 00 00 00
01 00 00 00
CE 56 00 00
10 27 00 00
7D 00 00 00
```

Esse formato já está implementado em `internal/legacyquery` e atendido por `cmd/query-server`.

## Envelope da resposta da consulta

`CPackageHeadSocket::GetRecvInfo` lê:

```text
uint32 remaining_length
uint32 unpacked_size
byte   payload[]
```

Todos os inteiros são little-endian nesse cliente.

- Tamanho total do pacote: `remaining_length + 4`.
- Se `unpacked_size == 0`, o `payload` é usado sem descompressão.
- Se `unpacked_size != 0`, o cliente usa zlib `uncompress()` com esse valor como capacidade de saída esperada.

O servidor atual envia a forma mais simples, sem compressão.

## Payload da resposta da consulta

O cliente constrói um `CMsgBuf` sobre o payload decodificado e consome:

```text
uint32 value0
uint32 value1
uint32 value2
uint32 value3
uint32 server_line_count
char   server_line_0[] + NUL
char   server_line_1[] + NUL
...
```

Os quatro primeiros inteiros são consumidos, mas o valor resultante não é utilizado pela rotina de recepção observada. A implementação mantém constantes compatíveis em vez de depender desse detalhe.

As strings terminam em `NUL`. `CMsgBuf::SetString` grava `strlen + 1`; `StringVal` lê uma string terminada em `NUL`.

## Como as strings retornadas viram `server_list.ini`

Fluxo confirmado:

1. A resposta retorna um vetor de strings.
2. `CParseServerListFile::WriteServerListFile` adiciona cada string sem alteração.
3. Após cada string, adiciona `\r\n`.
4. O resultado é salvo como `server_list.ini`.
5. O leitor genérico usa `strtok(..., "\r\n")`; portanto, cada string retornada representa uma linha lógica do arquivo.

## Gramática recuperada de `server_list.ini`

`FormLoginServers::LoadServerList` separa cada linha por `/` e consome três campos de texto seguidos de cinco inteiros decimais:

```text
server_id/district_key/address/port/type/open/load/priority
```

Semântica recuperada:

| Campo | Comportamento do cliente |
|---|---|
| `server_id` | copiado como ID do servidor de login escolhido |
| `district_key` | armazenado como distrito e também passado pelo caminho de localização do cliente |
| `address` | copiado diretamente como endereço do servidor de login |
| `port` | copiado diretamente como porta do servidor de login |
| `type` | `0` normal, `1` recomendado, `2` novo |
| `open` | `0` fechado, `1` aberto |
| `load` | quando aberto: `0` boa, `1` normal, `2` lotado |
| `priority` | menor valor vence entre servidores recomendados |

A rotina de recomendação filtra registros cujo `type == 1` e escolhe o menor `priority`. A interface mapeia `type == 1` para `server_recommand` (grafia original do cliente) e `type == 2` para `server_new`. Para estado, `open == 0` vira `server_stop`; quando `open == 1`, `load` 0/1/2 corresponde a `server_good`, `server_normal` e `server_hot`.

`server_issuer` é mantido separadamente pelo fluxo de login/seleção; ele não faz parte dos oito campos separados por `/`.

A gramática está implementada em `internal/legacyquery.ServerRecord`. `cmd/query-server` gera o registro a partir das variáveis `LEGACY_SERVER_*`.

## Framing da conexão membro/login

O cliente conecta ao endereço e porta selecionados e utiliza `LenSplitProtocol`, com um cabeçalho externo little-endian de 4 bytes contendo o tamanho do payload.

A requisição de login é montada em um buffer de trabalho fixo de `0x800` bytes.

### Cabeçalho do login

Os primeiros 21 bytes são:

```text
offset 0:    uint8  kind   = 2
offset 1:    uint32 family = 0x300
offset 5-20: 16 bytes zero
```

Primeiros cinco bytes:

```text
02 00 03 00 00
```

seguidos por 16 bytes zero.

### Blocos de texto variáveis

O builder armazena os textos como tamanho alinhado em 32 bits seguido de um bloco zerado, com os bytes reais no início:

```text
uint32 padded_size
byte   data[padded_size]
```

Regra recuperada:

```text
padded_size = align_up_16(strlen(value) + 16)
```

Mesmo uma string vazia recebe um bloco de 16 bytes. O cliente rejeita `device_id` maior que 64 bytes.

### Ordem completa dos campos

Após o cabeçalho de 21 bytes:

```text
padded-string account
padded-string password
padded-string login_string
uint32        login_type
uint8         in_stub
padded-string device_id
padded-string prop_md5
padded-string record_md5
padded-string name
uint32        info_length = 16
byte          info[16]
uint32        os_type
```

O pacote completo precisa caber no buffer recuperado de `0x800` bytes.

Wrappers do cliente foram observados usando `login_type` 4, 100, 200 e 201. O significado de provedor/conta desses números ainda não deve ser inferido sem prova adicional.

## Handshake e resposta de login

Já foram recuperados:

- `CLIENT_GET_VERIFY` (`0x00`)
- `SERVER_SET_VERIFY` (`0x01`)
- `CLIENT_RET_ENCODE` (`0x01`)
- `SERVER_SET_ENCODE` (`0x02`)
- `CLIENT_LOGIN` (`0x02`)
- `SERVER_LOGIN_SUCCESS` (`0x04`)

`SERVER_LOGIN_SUCCESS` começa com um prefixo fixo de **41 bytes** antes da lista de personagens. Cada personagem é serializado como uma sequência de parâmetros tipados, não como uma struct fixa simples.

A gramática desses registros está implementada em `internal/legacylogin`.

## Criação e escolha de personagem

`CLIENT_CREATE_ROLE` usa opcode `0x05`. O prefixo fixo recuperado possui 73 bytes:

```text
offset 0     uint8   opcode = 0x05
offset 1     uint32  field0
offset 5     uint16  name_utf16le[33]
offset 71    uint16  extra_parameter_count
offset 73    ...     parametros opcionais
```

`CLIENT_CHOOSE_ROLE` também foi recuperado e já é aceito pelo `member-server`.

## WORLD_INFO

Foram recuperados os dois sentidos:

- `CLIENT_WORLD_INFO` (`0x03`)
- `SERVER_WORLD_INFO` (`0x05`)

A resposta do servidor utiliza:

```text
uint8  opcode
uint16 field0
uint16 texto_utf16le[] + terminador NUL
```

## Bootstrap de cena

Os primeiros pacotes de inicialização da cena também foram recuperados:

- `SERVER_PROPERTY_TABLE` = `0x09`
- `SERVER_RECORD_TABLE` = `0x0A`
- `SERVER_ENTRY_SCENE` = `0x0B`

`PROPERTY_TABLE` e `RECORD_TABLE` aceitam uma forma mínima válida de 3 bytes:

```text
opcode
uint16 count = 0
```

`SERVER_ENTRY_SCENE` utiliza dois valores de 32 bits que formam a identidade da cena.

## Chave de cena `Scene-Member`

A análise do loader de `res\share\scenes.ini` confirmou que a identidade usada pelo cliente é:

```text
Scene-Member
```

A cena MOBA original encontrada no registro possui:

```text
Scene = 119
Member = 0
Config = ini\scene\moba_chiji
Clonable = 1
BattleWarID = 1
PreLoad = 1
```

Logo, a chave base conhecida pelo cliente é:

```text
119-0
```

O servidor pode emitir esse `ENTRY_SCENE` em modo experimental, mas só consideraremos a entrada no mapa confirmada após teste real com um APK 3.3.3 apontado para a VPS.

## Arquivo OBB / PCK0

O OBB principal usa o formato customizado `PCK0`, versão 15.

Cabeçalho observado:

```text
char[4] magic       = "PCK0"
uint32  version     = 15
uint16  flags       = 0
uint32  file_count  = 57879
uint32  data_offset = 4973871
uint8   reserved    = 0
```

Uma entrada de índice v15 começa com um `uint16` little-endian contendo o tamanho do registro e inclui:

```text
uint64 data_offset
uint32 unpacked_size
uint32 packed_size       # 0 = armazenado sem compressao
uint32 hash/check value
byte[5] metadata
char[] caminho terminado em NUL
```

O arquivo contém 57.879 recursos indexados; 57.690 usam compressão. A descompressão Deflate/zlib foi validada.

O parser/extrator independente está em `tools/pck0_extract.py`.

## Endpoints históricos de atualização encontrados

O cliente contém URLs antigas como:

```text
http://up-yxxg-us.snail.com/android36/patch
http://config-yxxg-us.snail.com/android36/version
http://up-yxxg-us.snail.com/android36/min
http://up-yxxg-us.snail.com/android36/res
http://up-yxxg-us.snail.com/android36/lng_patch
http://up-yxxg-us.snail.com/android36/bank
```

Elas são apenas evidência da arquitetura antiga. O servidor de preservação não deve depender dessa infraestrutura histórica.

## Próximos marcos

1. ✅ Recuperar a gramática de `server_list.ini`.
2. ✅ Gerar uma entrada de servidor própria.
3. ✅ Recuperar framing e campos de login.
4. ✅ Recuperar handshake e `SERVER_LOGIN_SUCCESS`.
5. ✅ Recuperar criação/seleção de personagem e `WORLD_INFO`.
6. ✅ Recuperar tabelas iniciais e `ENTRY_SCENE`.
7. ✅ Provar a chave `Scene-Member` e identificar `119-0`.
8. 🚧 Testar o fluxo completo com um APK 3.3.3 apontado para a VPS.
9. 🚧 Chegar de forma confirmada ao lobby/mapa original.
10. Recuperar negociação de sala/partida.
11. Substituir o laboratório de jogo pelo transporte de batalha UDP/KCP compatível.
12. Implementar sincronização, combate, loot, zona e persistência gradualmente.

## Nível de confiança

Salvo quando explicitamente marcado como hipótese, os valores deste documento foram observados no cliente 3.3.3 por configuração, símbolos nativos, disassembly ARM64 ou parsing determinístico dos arquivos. Nenhum código-fonte original do servidor foi utilizado.
