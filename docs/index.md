# svc-pedidos

API REST para consultar y registrar pedidos de una tienda ficticia. Es uno de los dos servicios de
ejemplo de la Internal Developer Platform del TFG y sigue la estructura estándar que genera la
plantilla de autoservicio.

## API

| Método y ruta | Descripción |
|---|---|
| `GET /` | Nombre del servicio, versión (SHA del commit) y entorno |
| `GET /api/pedidos` | Lista de pedidos; `?estado=pendiente` filtra por estado |
| `GET /api/pedidos/{id}` | Un pedido (`404` si no existe) |
| `POST /api/pedidos` | Registra un pedido (`201` + `Location`; `400` si el cuerpo no es válido) |
| `GET /healthz`, `GET /readyz` | Sondas de *liveness* y *readiness* |
| `GET /metrics` | Métricas en formato Prometheus |

Ejemplo de alta:

```bash
curl -X POST http://svc-pedidos.dev.localtest.me/api/pedidos \
  -H 'Content-Type: application/json' \
  -d '{"cliente":"cliente-010","lineas":[{"productoId":"P-002","cantidad":3}]}'
```

Los datos son sintéticos y viven en memoria: se pierden al reiniciar el pod. Las líneas
referencian productos de `svc-catalogo` por su identificador, sin llamada entre servicios.

## Métricas

| Métrica | Tipo | Etiquetas |
|---|---|---|
| `http_requests_total` | contador | `handler`, `method`, `code` |
| `http_request_duration_seconds` | histograma | `handler`, `method`, `code` |
| `app_build_info` | gauge (1) | `service`, `version`, `environment` |

Además, las métricas estándar del runtime de Go y del proceso.

## Configuración

| Variable | Por defecto | Uso |
|---|---|---|
| `PORT` | `8080` | Puerto HTTP |
| `APP_ENV` | `local` | Entorno que informa el servicio (`dev`, `staging`) |

## Despliegue

- Cada push a `main` ejecuta las pruebas y publica `ghcr.io/pablomartinvelasco/svc-pedidos:<sha>`.
- La configuración de despliegue está en
  [`idp-tfg/gitops/apps/svc-pedidos`](https://github.com/pablomartinvelasco/idp-tfg/tree/main/gitops/apps/svc-pedidos):
  Argo CD la sincroniza en `apps-dev` y `apps-staging`.
- Promoción: PR al repositorio de plataforma que cambia la etiqueta de imagen del overlay del entorno.

## Desarrollo local

```bash
go test ./...
go run .                      # http://localhost:8080
docker build -t svc-pedidos .
```
