# Análise de `server.txt`

Objetivo: permitir que o cliente 3.3.3 use um arquivo editável `server.txt` para escolher o host do servidor comunitário, com `142.44.162.24` como valor inicial.

## Descoberta atual

O cliente não depende apenas de `assets/config.ini`. O código ARM64 possui uma rotina separada que:

1. obtém um caminho-base por meio de um singleton interno;
2. concatena `server_query.ini` a esse caminho;
3. abre o arquivo como INI;
4. lê a seção `server_query` e as chaves `Addr`, `Port`, `Issuer` e `GameID`.

Isso é promissor porque `server_query.ini` tem 16 caracteres e pode ser substituído no binário por `server.txt` sem aumentar o tamanho da string original.

A análise já confirmou que o caminho-base vem de um objeto estático do runtime. A etapa atual procura a inicialização desse objeto no ARM64 para identificar o tipo da classe e o diretório real usado no Android.

Nenhum arquivo proprietário do jogo é incluído no repositório.
