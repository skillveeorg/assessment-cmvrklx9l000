# cx-claims-service

Microservicio de atencion al cliente y gestion de disputas transaccionales para comercio electronico.

## Arquitectura

El proyecto estructura el codigo Go separando responsabilidades en tres capas principales:

- `internal/api`: handlers `net/http` y ruteo de la API REST.
- `internal/service`: logica de negocio de reclamos y reembolsos.
- `internal/store`: acceso a datos (store en memoria).

El punto de entrada del servidor esta en `cmd/server`.

## Modelo de Reclamos

El ciclo de vida de una disputa contempla cuatro estados:

- `OPEN`: Disputa iniciada por el comprador pendiente de revision.
- `UNDER_REVIEW`: Reclamo evaluado por el equipo de CX apto para resolucion.
- `REFUNDED`: Reembolso procesado y acreditado.
- `REJECTED`: Disputa desestimada tras evaluacion operativa.

Unicamente los casos en estado `UNDER_REVIEW` son elegibles para procesar reembolsos manuales.

## Ejecucion Local

Para iniciar el servidor HTTP en el puerto 8080:

```bash
go run ./cmd/server
```


## Pruebas

La suite de pruebas automatizadas del servicio se ejecuta con fixtures en memoria:

```bash
go test ./...
```

