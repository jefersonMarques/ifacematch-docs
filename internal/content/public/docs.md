# iFaceMatch — Documentação pública de integração

Fonte canônica: https://docs.ifacematch.com/
Atualizado: 2026-11-12. Idioma: pt-BR.

## Arquitetura

Customer Hosted: a API/Server de biometria opera na infraestrutura autorizada do cliente, enquanto Portal e Central gerenciam licenças e distribuição. Não enviar chaves privadas ou dados biométricos à Central por padrão.

## Autorização

Autenticação REST da aplicação: cabeçalho `Authorization: Bearer <application-key>`. Credenciais de aplicação ficam no backend, idealmente em arquivo com permissão 0600 ou gerenciador de segredos. Nunca usar esta chave no SDK Web, HTML, localStorage ou URLs.

## API REST confirmada pelos exemplos do produto

Base HTTPS: `https://face.example.com` (domínio da instalação autorizada).

| Operação | Criar | Consultar |
| --- | --- | --- |
| Prova de vida independente | `POST /v1/liveness/checks` com `{}` | `GET /v1/liveness/checks/{checkId}` |
| Cadastro de referência | `POST /v1/enrollments` com `{"subjectId":"person_123"}` | `GET /v1/enrollments/{checkId}` |
| Verificação facial 1:1 | `POST /v1/verifications` com `{"subjectId":"person_123"}` | `GET /v1/verifications/{checkId}` |

As respostas de criação HTTP 201 incluem `id` e `verificationUrl`. A URL é efêmera, HTTPS, com caminho `/verify.html` e fragmento de token. O navegador executa a captura pelo SDK. O backend consulta a decisão efetiva usando o `checkId`. O sucesso de HTTP 201 não equivale a aprovação biométrica.

Exemplo de requisição backend:

```bash
curl --request POST \
  "$IFACEMATCH_API_ORIGIN/v1/liveness/checks" \
  --header "Authorization: Bearer $IFACEMATCH_API_KEY" \
  --header "Content-Type: application/json" \
  --data '{}'
```

O trecho acima serve de demonstração. Em produção, evite exportar credenciais em ambiente compartilhado ou imprimir tokens em logs/shell history.

## Prévia: cadastro por envio de imagem (ainda indisponível em produção)

A implementação encontra-se no [Server PR #56](https://github.com/jefersonMarques/ifacematch-server/pull/56), em Draft. A distribuição 0.1.11 atualizou somente o Console, portanto **não fornece esses endpoints**; a ativação exige Server compatível e Secure Core >= 0.2.2. Esta seção é contrato em desenvolvimento, não API comercial estável.

- `POST /v1/enrollments/images`: Bearer com `enrollment:create`, `multipart/form-data`, exatamente `subjectId` (1–128 caracteres) e `image` (JPEG/PNG até 4 MiB, 80–4096 px por dimensão e máximo de 8 megapixels).
- `GET /v1/subjects/{subjectId}/enrollment`: requer `checks:read`, retorna metadados da referência.
- `GET /v1/subjects/{subjectId}/enrollment/image`: requer `enrollment:image:read`, retorna JPEG normalizado com `Cache-Control: no-store`.
- `DELETE /v1/subjects/{subjectId}/enrollment`: revogação da referência pela rota existente.
- Sem rosto, com vários rostos ou qualidade insuficiente: HTTP 422; referência duplicada: 409; Secure Core incompatível: 503. Importar uma foto não é prova de vida nem autenticação da identidade.

Exemplo planejado para backend autorizado:

```bash
curl --request POST \
  "$IFACEMATCH_API_ORIGIN/v1/enrollments/images" \
  --header "Authorization: Bearer $IFACEMATCH_API_KEY" \
  --form "subjectId=driver-123" \
  --form "image=@/secure/path/reference.jpg;type=image/jpeg"
```

Documentação completa: https://docs.ifacematch.com/upload-image; contrato **exclusivamente de prévia**: https://docs.ifacematch.com/openapi-preview.json. A especificação estável `/openapi.json` continua sem essas operações até a distribuição comercial.

## SDK Web

O SDK instalado serve `verify.html` e dependências versionadas no domínio autorizado. O frontend da aplicação pede ao próprio backend uma URL temporária e redireciona o navegador. `/api/identity/start` é uma rota ilustrativa do backend do integrador e não pertence à API nativa.

## Similaridade facial

A comparação é 1:1. No Console, administradores com MFA recente podem atribuir limite por aplicação. O valor 0 herda o mínimo global; valores personalizados não podem ser inferiores ao mínimo global nem superiores a 0.65. Isto é score técnico, não porcentagem de certeza. O controle não altera liveness. O endpoint interno `POST /console/v1/applications/{applicationId}/match-threshold` não é público.

## Instalação

Ubuntu Server 24.04 amd64 com systemd, PostgreSQL 16 e HTTPS confiável. Usar somente release autorizada e assinada. Perfil compartilhado exige que o administrador configure proxy reverso sem sobrescrever serviços existentes.

## Releases

- Pacote `0.1.7`, implantação piloto privada em 05/11/2026: Console 0.1.4, Server 0.1.6 mantido. SHA-256: `65358027c374a794f9a0201df229469da49e0b844a59486303e27df1a434b9a5`.
- Pacote `0.1.6`, implantação piloto privada em 04/11/2026: Server 0.1.6. SHA-256: `608197ec0bf762efc061c2dd6a9b047c293932c37a2b4df507a65915c9278c16`.
- Assinatura da distribuição: Ed25519, autoridade `ifm_distribution_prod_v1`.
- Os pacotes do piloto não estão publicados para download comercial geral.

## Upgrade

Validar a autorização da release, SHA-256, assinatura, versões e compatibilidade; fazer backup; aplicar apenas componentes alterados; validar health e login e executar smoke 1:1 autorizado. Nunca executar novamente instalador de primeira instalação em cliente ativo. Se houver estado parcial, inspecionar antes de repetir. `ifacematchctl upgrade --latest` é uma proposta futura, não um comando disponível hoje.

## Referências

- https://docs.ifacematch.com/api
- https://docs.ifacematch.com/sdk
- https://docs.ifacematch.com/installation
- https://docs.ifacematch.com/releases
- https://docs.ifacematch.com/upgrade
