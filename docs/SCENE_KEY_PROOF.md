# Prova da chave composta de cena

`SERVER_ENTRY_SCENE` formata dois valores `uint32` como `%u-%u` e procura o texto resultante no registro de cenas do cliente. O arquivo `res\share\scenes.ini` define tanto `Scene` (ID da cena) quanto `Member` (número do servidor de cena).

## Ordem comprovada

O caminho `SceneBaseModule::LoadSceneBaseInfo` abre `share/scenes.ini`, percorre cada seção numérica e converte o nome da seção para inteiro. Em seguida, lê o valor `Member` da mesma seção.

Imediatamente antes da chamada virtual que adiciona a cena, o cliente ARM64 coloca:

```text
w1 = secao numerica do INI / Scene ID
w2 = valor Member
x3 = Script
x4 = Config
```

Portanto, o caminho de adição recebe `Scene` primeiro e `Member` depois.

Em outro ponto, o cliente representa a mesma identidade em um par de 32 bits e formata o primeiro componente seguido do segundo com `%u-%u`.

Combinando isso com `GameReceiver::ServerEntryScene`, a chave de lookup em execução é:

```text
Scene-Member
```

não `Member-Scene`.

## Primeira cena controlada

O registro arquivado contém o mapa MOBA original:

```text
section/Scene = 119
Member = 0
Config = ini\scene\moba_chiji
Clonable = 1
BattleWarID = 1
PreLoad = 1
```

Logo, a chave base pré-carregada conhecida pelo cliente é:

```text
119-0
```

O servidor de compatibilidade usa `119` / `0` apenas como experimento **opt-in** de `SERVER_ENTRY_SCENE`. O envio continua desligado até `MEMBER_SEND_ENTRY_SCENE=true` ou até o perfil `scripts/enable-legacy-flow.sh` ser ativado.

Isso separa o que já foi provado por análise estática do que ainda precisa de confirmação em execução: chegar ao lobby/mapa continua dependendo de um teste real com cliente próprio apontado para a VPS.
