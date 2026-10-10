# Survival Heroes Legacy Server

Reimplementação comunitária e independente do servidor de **Survival Heroes**, criada para preservação e para permitir que clientes antigos voltem a se conectar sem depender da infraestrutura oficial encerrada.

> **Status atual:** a base de compatibilidade do cliente Android **3.3.3** já está bem avançada. O bootstrap de servidores, enquadramento TCP, login, criação/seleção de personagem, `WORLD_INFO`, tabelas iniciais e a chave de entrada de cena foram recuperados. O primeiro mapa MOBA conhecido pelo cliente foi identificado como **`119-0`**. Ainda falta confirmar esse fluxo completo em um APK de teste apontado para a VPS e, depois disso, reconstruir lobby, salas e protocolo de batalha.

## Objetivos

- Usar como alvo principal a linha final conhecida do cliente Android, começando pela **3.3.3**.
- Reimplementar o comportamento do servidor a partir da análise do cliente, sem distribuir código-fonte original do servidor nem arquivos proprietários do jogo.
- Priorizar partidas entre jogadores reais; bots, caso sejam adicionados no futuro, devem ser opcionais e claramente identificados.
- Facilitar hospedagem própria em VPS pequena/aaPanel usando Docker Compose.
- Documentar o protocolo para que a preservação do jogo não dependa de uma única pessoa.

## Progresso atual

### Já recuperado do cliente original 3.3.3

- Endpoint original de consulta/listagem de servidores.
- Requisição exata de **28 bytes little-endian** usada na porta TCP **88**.
- Formato da resposta de consulta, incluindo dados brutos/zlib e payload terminado em `NUL`.
- Geração do `server_list.ini`.
- Formato da linha de servidor: `server_id/district/address/port/type/open/load/priority`.
- Semântica de tipo, estado, carga e prioridade de servidor.
- Enquadramento `LenSplitProtocol` usado pela conexão de login/membro.
- Estrutura e ordem dos campos de `CLIENT_LOGIN`.
- Handshake `CLIENT_GET_VERIFY` / `SERVER_SET_VERIFY`.
- `CLIENT_RET_ENCODE` / `SERVER_SET_ENCODE`.
- Prefixo e estrutura de `SERVER_LOGIN_SUCCESS`.
- Gramática genérica dos registros de personagem.
- Estrutura de `CLIENT_CREATE_ROLE`.
- Estrutura de `CLIENT_CHOOSE_ROLE`.
- Estrutura de `CLIENT_WORLD_INFO` e `SERVER_WORLD_INFO`.
- `SERVER_PROPERTY_TABLE` e `SERVER_RECORD_TABLE`, incluindo pacotes vazios válidos.
- `SERVER_ENTRY_SCENE`.
- Prova de que a chave de cena é **`Scene-Member`**.
- Cena MOBA original identificada no cliente como **`119-0`** (`moba_chiji`).
- Formato do arquivo principal do OBB: **PCK0 v15**.
- Ferramenta própria para listar/extrair PCK0 em `tools/pck0_extract.py`.
- Arquitetura de batalha identificada como canais UDP nativos com **KCP** no canal confiável/sequenciado.
- Runtime identificado como C++ nativo com componentes Cocos2d-x, Lua e PhysX; o jogo não é Unity.

Os detalhes técnicos continuam documentados em `docs/`.

## Componentes do servidor

- **`query-server`** — implementação compatível com a consulta original TCP/88 e geração da lista de servidores.
- **`member-server`** — serviço de compatibilidade do protocolo original na TCP/7001. Já entende o fluxo inicial de handshake, login, personagem, informações de mundo e entrada de cena recuperados.
- **`gateway`** — laboratório HTTP da fase inicial; não representa o protocolo final do cliente original.
- **`game-server`** — laboratório TCP/JSON da fase inicial; não representa ainda o protocolo de batalha final.
- **Docker Compose** para subir os serviços.
- Testes e build automáticos pelo GitHub Actions.
- Ferramentas de análise de APK/XAPK/OBB.
- Patcher de host do cliente em `tools/patch_client.py`.

## Início rápido

```bash
cp .env.example .env
sh scripts/setup.sh
sh scripts/smoke-test.sh
```

Serviços padrão:

- Consulta de servidores compatível com o cliente original: `tcp://localhost:88`
- Serviço de login/membro: `tcp://localhost:7001`
- Gateway de desenvolvimento: `http://localhost:8080`
- Socket de jogo de desenvolvimento: `tcp://localhost:7000`

Para publicar sua VPS na tela original de seleção de servidor:

```env
LEGACY_SERVER_HOST=seu.ip.ou.dominio
LEGACY_SERVER_PORT=7001
```

Também existe um configurador simples:

```bash
sh scripts/configure-public-host.sh SEU_IP_OU_DOMINIO
```

## Ativar o fluxo legado recuperado

Para um teste controlado com um cliente 3.3.3 de sua propriedade:

```bash
sh scripts/enable-legacy-flow.sh
sh scripts/setup.sh
```

Esse perfil ativa experimentalmente:

```text
CLIENT_GET_VERIFY
→ SERVER_SET_VERIFY

CLIENT_RET_ENCODE
→ SERVER_SET_ENCODE

CLIENT_LOGIN
→ SERVER_LOGIN_SUCCESS

CLIENT_WORLD_INFO
→ SERVER_WORLD_INFO

CLIENT_CHOOSE_ROLE
→ SERVER_PROPERTY_TABLE vazio
→ SERVER_RECORD_TABLE vazio
→ SERVER_ENTRY_SCENE 119-0
```

Isso significa que a estrutura necessária para tentar chegar ao mapa original está implementada, **mas ainda não afirmamos que o lobby/mapa funciona até o teste real com o APK modificado ser concluído**.

## Quanto falta?

Para o **primeiro teste de entrada no cliente original**, falta relativamente pouco: apontar o APK para a VPS, executar o fluxo e observar qual é o primeiro pacote ou estado que ainda bloqueia a entrada.

Para o jogo ficar **realmente jogável online**, ainda existem blocos grandes de trabalho:

1. Confirmar login → personagem → cena no APK real.
2. Reconstruir lobby, perfil e estado inicial do jogador.
3. Recuperar criação/entrada em sala e matchmaking original.
4. Implementar transporte de batalha UDP/KCP.
5. Sincronizar jogadores, movimento e entidades.
6. Implementar combate, armas/habilidades, loot e inventário.
7. Implementar zona fechando, morte, vitória e encerramento da partida.
8. Adicionar persistência de conta/personagem e demais sistemas necessários.

Ou seja: **a camada de descoberta inicial avançou bastante, mas o servidor completo ainda tem trabalho pela frente**. A vantagem é que agora já saímos da fase de procurar “onde o cliente conecta” e estamos chegando na fase de fazer o jogo efetivamente avançar.

## Arquivos do cliente

Não envie APK, XAPK ou OBB proprietários para este repositório. As ferramentas de análise e repacotamento trabalham com um cliente fornecido separadamente pelo próprio usuário.

## Roadmap

- [x] Inspecionar o Survival Heroes 3.3.3 e seus arquivos principais.
- [x] Recuperar o protocolo TCP/88 de consulta de servidores.
- [x] Recuperar o formato exato da lista de servidores.
- [x] Recuperar o framing e a requisição de login/membro.
- [x] Recuperar handshake e `SERVER_LOGIN_SUCCESS`.
- [x] Recuperar criação e seleção de personagem.
- [x] Recuperar `WORLD_INFO`.
- [x] Recuperar tabelas iniciais de propriedade/registro.
- [x] Recuperar `ENTRY_SCENE` e provar a chave `Scene-Member`.
- [x] Identificar a cena MOBA `119-0`.
- [ ] Testar o fluxo completo com APK 3.3.3 apontado para a VPS.
- [ ] Chegar de forma confirmada ao lobby/mapa original.
- [ ] Recuperar salas e matchmaking.
- [ ] Implementar UDP/KCP de batalha.
- [ ] Implementar sincronização, combate, loot e zona.
- [ ] Implementar persistência e sistemas restantes.

## Nota de preservação

Este projeto é uma reimplementação independente para interoperabilidade e preservação. Ele não contém o código-fonte original do servidor e não deve redistribuir APK, OBB, marcas, artes ou outros dados proprietários do jogo.
