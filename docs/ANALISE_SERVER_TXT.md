# Análise de `server.txt`

Objetivo: permitir que o cliente 3.3.3 use um arquivo editável `server.txt` para escolher o host do servidor comunitário, com `142.44.162.24` como valor inicial.

Antes de alterar o binário, o projeto verifica como `assets/config.ini` é localizado pelo runtime nativo. Se o mecanismo já priorizar um diretório gravável/externo, o patch pode ser simples e robusto. Caso contrário, será necessário um pequeno redirecionamento de carregamento dentro do cliente.

Nenhum arquivo proprietário do jogo é incluído no repositório.
