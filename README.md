# iFaceMatch Docs

Documentação técnica pública independente para o iFaceMatch Customer Hosted.

- Produção prevista: **https://docs.ifacematch.com**
- Repositório: [jefersonMarques/ifacematch-docs](https://github.com/jefersonMarques/ifacematch-docs)
- Backend: Go + [templ](https://templ.guide/) + [HTMX](https://htmx.org/)
- Distribuição: **um binário Linux amd64** (páginas e assets incorporados com `embed.FS`)
- Nenhum banco de dados, container, Node.js ou runtime frontend é necessário no servidor.
- O Server biométrico, o Console e a Central permanecem independentes.

## Estrutura

```text
cmd/docs/                   Entry point, health endpoint
internal/site/              HTTP routes, security headers, HTMX search
internal/views/             templ layout and navigation
internal/content/           Version-controlled documentation and embedded assets
deploy/systemd/             Isolated documentation service
deploy/nginx/               Dedicated docs.ifacematch.com virtual host
.github/workflows/          Tests and GitHub binary Releases
```

## Desenvolvimento

Requer Go 1.25.x. O compilador `templ` usa uma versão fixa no build.

```bash
make test
make run
curl --fail http://127.0.0.1:8091/healthz
```

Páginas completas são geradas no servidor e indexáveis sem JavaScript. HTMX apenas melhora a busca (`GET /search`); o formulário também funciona com uma resposta HTML completa quando HTMX não está disponível. Os ativos JavaScript são distribuídos com a aplicação, sem CDN.

## API pública do site

| Rota | Função |
| --- | --- |
| `/` | Introdução e início rápido |
| `/api` | Endpoints REST |
| `/upload-image` | Prévia de envio de fotografia facial; não disponível na release comercial 0.1.11 |
| `/sdk` | SDK Web |
| `/installation` | Instalação |
| `/releases` | Histórico de versões |
| `/upgrade` | Upgrade e rollback |
| `/search?q=` | Busca SSR/HTMX |
| `/openapi.json` | OpenAPI 3.1 |
| `/openapi-preview.json` | OpenAPI 3.1 separada com endpoints de desenvolvimento não distribuídos |
| `/llms.txt`, `/llms-full.txt`, `/docs.md` | Documentação legível por agentes |
| `/robots.txt`, `/sitemap.xml` | Descoberta e indexação |
| `/healthz` | Health check local |

As URLs antigas com extensão `.html` recebem redirecionamento HTTP 301 para as rotas canônicas. As versões com barra final, como `/api/`, também redirecionam para `/api`. As consultas `/search` são marcadas como `noindex`.

As rotas do Portal e da Central **não** são expostas neste serviço. O exemplo `/example.html` é entregue como código-fonte em texto, e não como fluxo biométrico executável.

## GitHub Releases

O workflow `Documentation CI` valida cada Pull Request. O workflow `Signed-source binary release` roda somente quando uma tag semântica `vMAJOR.MINOR.PATCH` é enviada.

```bash
git tag v0.1.1
git push origin v0.1.1
```

A Release inclui somente:

- `ifacematch-docs-linux-amd64`: executável estático contendo todos os arquivos.
- `SHA256SUMS`: hash de integridade verificável antes da instalação.

A Release **não realiza deploy automaticamente**. O binário não é assinado criptograficamente: SHA-256 verifica integridade comparada ao checksum publicado, não autentica por si só a origem. Para um nível superior de segurança, planejar assinatura de artefatos e verificação independente antes de automatizar upgrades.

## Servidor e isolamento

1. Configurar DNS `docs.ifacematch.com` para o servidor escolhido.
2. Configurar HTTPS e o certificado válido do domínio antes de ativar o bloco Nginx em `deploy/nginx/docs.ifacematch.com.conf`.
3. Criar `/opt/ifacematch-docs/bin` e instalar o binário verificado em `/opt/ifacematch-docs/bin/ifacematch-docs`.
4. Copiar a unit de `deploy/systemd/ifacematch-docs.service` para `/etc/systemd/system/` e executar `systemctl daemon-reload`, `systemctl enable --now ifacematch-docs`.
5. Ativar o vhost, validar `nginx -t` e recarregar o Nginx.
6. Verificar `/healthz`, as sete páginas (incluindo a prévia de envio de imagem), a busca, `/openapi.json`, `/openapi-preview.json` e os arquivos para IA.

Para atualizar uma instalação existente, baixar o executável e `SHA256SUMS` da Release aprovada e executar `deploy/install-release.sh` como root. O script verifica o checksum, preserva o binário anterior e tenta rollback se o serviço ou o health check falhar. Não use este script para a instalação inicial.

## SEO e migração

- Cada página tem HTML indexável, título, descrição, canonical no novo domínio e navegação interna.
- O sitemap aponta para `docs.ifacematch.com`, com versão Markdown, OpenAPI e `llms.txt` para agentes.
- **Somente depois** de validar o novo host: redirecionar a antiga rota `portal.ifacematch.com/docs.html` e demais URLs antigas com 301 para as novas URLs correspondentes. Não remover a documentação anterior antes da verificação.
- O repositório é público; não publicar chaves, modelos proprietários, dados de clientes nem artefatos internos.

## Licenças

O código e a documentação própria do iFaceMatch não possuem licença open source declarada neste repositório. O HTMX é redistribuído com os termos da sua licença em `THIRD_PARTY_LICENSES/htmx-LICENSE.txt`.
