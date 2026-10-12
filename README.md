# svc-pedidos

Pedidos de la tienda de demostración: servicio de ejemplo de la
[IDP del TFG](https://github.com/pablomartinvelasco/idp-tfg).

- API: `GET /api/pedidos`, `GET /api/pedidos/{id}`, `POST /api/pedidos`; métricas en `/metrics`.
- Imagen: `ghcr.io/pablomartinvelasco/svc-pedidos:<sha>` (CI en cada push a `main`).
- Despliegue: [`idp-tfg/gitops/apps/svc-pedidos`](https://github.com/pablomartinvelasco/idp-tfg/tree/main/gitops/apps/svc-pedidos).
- Documentación: [`docs/index.md`](docs/index.md) (publicada en el portal con TechDocs).
