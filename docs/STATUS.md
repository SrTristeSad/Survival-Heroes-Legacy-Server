# Status do projeto

## Alvo atual

- Jogo: **Survival Heroes**
- Pacote Android: `com.snailgamesusa.survival`
- Linha de cliente estudada: **3.3.3**
- Build APK arquivada observada: `103` (21 de março de 2024)
- Existem também arquivos XAPK/OBB completos de aproximadamente 1 GB da linha 3.3.3, usados somente para pesquisa e preservação.

## O que já está confirmado

### Bootstrap e seleção de servidor

- Requisição original de consulta TCP/88 recuperada.
- Resposta e framing recuperados.
- Formato da lista de servidores recuperado.
- Geração de registro compatível apontando para uma VPS própria.

### Login e personagem

- `LenSplitProtocol` recuperado.
- `CLIENT_GET_VERIFY` / `SERVER_SET_VERIFY`.
- `CLIENT_RET_ENCODE` / `SERVER_SET_ENCODE`.
- `CLIENT_LOGIN`.
- `SERVER_LOGIN_SUCCESS`.
- Estrutura dos registros de personagem.
- `CLIENT_CREATE_ROLE`.
- `CLIENT_CHOOSE_ROLE`.

### Entrada de mundo/cena

- `CLIENT_WORLD_INFO` e `SERVER_WORLD_INFO`.
- `SERVER_PROPERTY_TABLE`.
- `SERVER_RECORD_TABLE`.
- `SERVER_ENTRY_SCENE`.
- Ordem da chave de cena confirmada como `Scene-Member`.
- Mapa MOBA original identificado como `119-0` (`moba_chiji`).

### Ferramentas e infraestrutura

- Código Go compila sem dependências externas desnecessárias.
- Docker Compose validado.
- Testes automáticos ativos.
- Parser/extrator PCK0 v15 funcional.
- Patcher de host do cliente disponível.
- Fluxo experimental completo pode ser habilitado por `scripts/enable-legacy-flow.sh`.

## Próximo teste crítico

O próximo passo não é mais apenas análise estática: é executar um APK 3.3.3 de teste apontado para a VPS e verificar até onde o cliente chega com o fluxo já recuperado.

O objetivo imediato é confirmar na prática:

```text
seleção de servidor
→ handshake
→ login
→ personagem
→ WORLD_INFO
→ PROPERTY_TABLE / RECORD_TABLE
→ ENTRY_SCENE 119-0
```

Qualquer novo pacote enviado pelo cliente depois desse ponto passa a indicar exatamente qual subsistema deve ser reconstruído em seguida.

## O que ainda falta para ficar jogável

Ainda faltam partes grandes do servidor final:

- lobby e estado completo do jogador;
- salas e matchmaking original;
- protocolo de batalha;
- transporte UDP/KCP completo;
- sincronização de jogadores e entidades;
- movimento autoritativo;
- combate e habilidades;
- loot e inventário;
- zona segura/fechamento do mapa;
- morte, vitória e finalização de partida;
- persistência de conta e personagem;
- demais sistemas sociais e progressão conforme forem necessários.

### Resumo

**Para o primeiro teste dentro do cliente original: estamos perto.**

**Para recriar o jogo completo online: ainda falta bastante.**

A diferença é que a camada inicial de descoberta já avançou muito: agora o trabalho começa a migrar de engenharia reversa de conexão para reconstrução efetiva do comportamento do servidor.
