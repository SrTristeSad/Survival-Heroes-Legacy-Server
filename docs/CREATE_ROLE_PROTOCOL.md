# Requisição `CREATE_ROLE` do Survival Heroes 3.3.3

O caminho `GameSender::CreateRole` do cliente arquivado foi recuperado o suficiente para confirmar o prefixo fixo da requisição sem adivinhar o significado dos campos.

## Estrutura fixa

A requisição usa opcode de cliente `0x05` e possui tamanho fixo mínimo de 73 bytes:

```text
offset 0     uint8   opcode = 0x05
offset 1     uint32  field0
offset 5     uint16  name_utf16le[33]
offset 71    uint16  extra_parameter_count
offset 73    ...     parametros genericos opcionais
```

A área do nome é gerada a partir do argumento wide-string e limitada a 33 unidades UTF-16 no cliente ARM64 analisado. O segundo argumento do lado de script precisa usar o tipo wide-string `7` do cliente.

O primeiro argumento é convertido para 32 bits e armazenado no offset `1`. Seu significado de alto nível permanece intencionalmente sem nome até que um caller ou caminho de dados prove sua função.

O sender grava `argument_count - 2` no contador de parâmetros extras de 16 bits. Parâmetros adicionais são serializados após o prefixo fixo de 73 bytes pela mesma gramática genérica de parâmetros usada em outras partes do protocolo.

`internal/legacylogin/create_role.go` valida esse prefixo. `cmd/member-server` registra apenas metadados estruturais e não imprime o nome escolhido pelo jogador.

## Situação atual

A etapa seguinte, `CLIENT_CHOOSE_ROLE`, também já foi recuperada. Depois da escolha de personagem, o fluxo experimental segue por `WORLD_INFO`, tabelas iniciais e `SERVER_ENTRY_SCENE`.
