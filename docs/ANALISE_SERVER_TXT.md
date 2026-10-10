# Análise de `server.txt`

Objetivo: permitir que o cliente 3.3.3 use um arquivo editável `server.txt` para escolher o host do servidor comunitário, com `142.44.162.24` como valor inicial.

## Uso atual

O projeto já possui `client-config/server.txt` com o IP atual:

```text
142.44.162.24
```

`scripts/build-client-vps.sh` usa esse arquivo automaticamente quando nenhum host é passado na linha de comando. Portanto, para o teste atual, não é necessário editar o IP nem informar argumento ao script.

No APK gerado hoje, o endereço também é gravado em `assets/config.ini` como fallback, para o primeiro teste funcionar de forma determinística.

## Descoberta para a futura troca sem rebuild

O cliente não depende apenas de `assets/config.ini`. O código ARM64 possui uma rotina separada que:

1. obtém um caminho-base por meio de um objeto interno do runtime;
2. concatena `server_query.ini` a esse caminho;
3. abre o arquivo como INI;
4. lê a seção `server_query` e as chaves `Addr`, `Port`, `Issuer` e `GameID`.

Isso mostra que o cliente original já possuía um mecanismo de configuração externa de servidor. A string `server_query.ini` pode tecnicamente ser substituída por `server.txt` sem aumentar o espaço ocupado no binário.

O diretório Android exato usado por esse mecanismo ainda não foi promovido como fato confirmado. Por isso, a leitura direta de `server.txt` em tempo de execução fica para uma etapa posterior; não é necessária para testar o servidor atual.

Nenhum arquivo proprietário do jogo é incluído no repositório.
