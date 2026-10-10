# Status da recuperação de `SERVER_LOGIN_SUCCEED`

A análise independente do cliente 3.3.3 confirmou o caminho `GameReceiver::ServerLoginSucceed`.

## Confirmado

- Opcode do servidor: `0x04`.
- Prefixo fixo: **41 bytes**.
- `role_count` em offset `37`.
- Registros de personagem começam em offset `41` quando `role_count > 0`.
- Cada personagem começa com `uint16 parameter_count`.
- Os parâmetros usam tags tipadas `2` a `8`.
- O encoder correspondente já existe em `internal/legacylogin/role_record.go`.
- Um pacote com `role_count=0` é válido no parser e pode ser usado para alcançar o caminho original de criação de personagem sem inventar dados de personagem.

## Situação atual

A recuperação de `SERVER_LOGIN_SUCCEED` deixou de ser um gargalo principal. O fluxo já avançou para criação/seleção de personagem, `WORLD_INFO` e entrada de cena.

Nenhum APK, OBB ou outro recurso proprietário do cliente é incluído neste repositório.
