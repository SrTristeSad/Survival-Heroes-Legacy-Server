# Protocolo de registros de personagem no login — Survival Heroes 3.3.3

Este documento registra a gramática da lista de personagens consumida pelo caminho `GameReceiver::ServerLoginSucceed` do cliente Android 3.3.3 arquivado.

## Prefixo de login-success

A mensagem do servidor usa opcode `0x04`. O cliente exige um prefixo fixo de 41 bytes. Um `uint32` little-endian no offset `37` informa quantos registros de personagem vêm imediatamente depois do prefixo.

Um `role_count=0` é válido no parser. Por isso, o servidor de compatibilidade pode emitir um pacote controlado sem personagens para testar o caminho original de criação de personagem sem inventar dados.

## Registro de personagem

Cada personagem começa com:

```text
uint16 parameter_count
```

Depois vêm `parameter_count` parâmetros tipados. Cada parâmetro começa com uma tag de um byte. O receiver 3.3.3 aceita exatamente tags `2` até `8`:

| Tag | Valor no fio |
|---:|---|
| `2` | `uint32` |
| `3` | `uint64` |
| `4` | `uint32` |
| `5` | `uint64` |
| `6` | `uint32 byte_length` + string de bytes terminada em NUL; o tamanho inclui o NUL final |
| `7` | `uint32 byte_length` + string UTF-16LE terminada em NUL; o tamanho é em bytes e inclui o `0x0000` final |
| `8` | dois valores `uint32` consecutivos |

Outras tags entram no caminho de erro `(GameReceiver::ServerLoginSucceed)role parameter type error`.

O encoder está implementado em `internal/legacylogin/role_record.go` e usa nomes neutros para os parâmetros até que o significado de alto nível de cada campo seja provado.

## Materialização observada no cliente

Depois de interpretar o vetor genérico de parâmetros, o cliente copia alguns índices para um objeto interno de resumo do personagem.

Observações confirmadas:

- índice `0`, quando tag `2`, vai para um `uint32` no offset `0` do objeto;
- índice `1`, quando tag `2`, vai para um `uint32` no offset `4`;
- índice `2`, quando tag `7`, é copiado como string wide no offset `8`;
- índice `3`, quando tag `7`, é copiado como string wide no offset `0x68`;
- índice `4`, quando tag `2`, é armazenado no offset `0xF8` quando existem parâmetros suficientes;
- índice `5`, quando tag `5`, é armazenado como valor de oito bytes no offset `0x100`;
- índice `7`, quando tag `6`, é copiado como string de bytes no offset `0xC8` quando presente.

Os significados de negócio desses campos continuam sem nome intencionalmente. Saber o tipo no fio e a posição em memória não é evidência suficiente para chamar um valor de ID, nível, nome ou outro dado da conta.

## Experimento atual

`MEMBER_SEND_LOGIN_SUCCESS=true` faz `cmd/member-server` responder a um `CLIENT_LOGIN` válido com o prefixo recuperado de 41 bytes e `role_count=0`.

A criação e seleção de personagem já foram recuperadas depois desse ponto. O fluxo experimental atual continua por `CLIENT_CREATE_ROLE` / `CLIENT_CHOOSE_ROLE`, `WORLD_INFO` e bootstrap de cena.
