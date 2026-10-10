# Trabalho de compatibilidade com o cliente

Cliente alvo: Survival Heroes 3.3.3 (`com.snailgamesusa.survival`).

Este repositório intencionalmente **não** inclui APK, OBB ou recursos proprietários. O operador fornece localmente um pacote de cliente obtido de forma legítima.

## Fase A — inventário estático

1. Preservar APK/XAPK/OBB sem modificações e registrar SHA-256.
2. Executar as ferramentas de análise do projeto sobre o cliente.
3. Inventariar permissões, activities, bibliotecas `.so` e runtime do jogo.
4. Identificar endpoints fixos de bootstrap, login e conteúdo.
5. Determinar serialização e protocolos usados pelo cliente.
6. Separar fatos confirmados de hipóteses.

## Fase B — observação de rede

Executar somente com cliente/dispositivo controlado pelo próprio testador. Registrar consultas DNS, destinos de conexão e características TLS/plaintext.

Não interceptar credenciais de terceiros. O objetivo é reconstruir a interface histórica necessária para interoperabilidade, não acessar qualquer serviço privado sobrevivente.

## Fase C — compatibilidade independente

Implementar as semânticas observadas em serviços próprios e documentar os campos como confirmados, inferidos ou desconhecidos.

Código provisório deve permanecer separado do protocolo confirmado até que o comportamento seja validado.

## Marcos de compatibilidade

Primeiro marco:

```text
cliente 3.3.3
→ bootstrap local
→ login local
→ personagem
→ entrada confirmada no lobby/mapa original
```

Segundo marco:

```text
dois clientes reais
→ matchmaking local
→ mesma partida
→ movimento visível nos dois clientes
```

Terceiro marco:

```text
partida completa
→ combate
→ loot
→ fechamento da zona
→ morte/vitória
→ encerramento correto da sessão
```

## Situação atual

Já foram recuperados bootstrap TCP/88, lista de servidores, framing de membro/login, handshake, login-success, registros de personagem, criação/seleção de personagem, `WORLD_INFO`, tabelas iniciais e `ENTRY_SCENE`.

A chave de cena foi confirmada como `Scene-Member` e o primeiro mapa MOBA conhecido pelo cliente é `119-0`.

O próximo passo crítico é validar esse fluxo em execução real com um cliente 3.3.3 do próprio testador apontado para a VPS.
