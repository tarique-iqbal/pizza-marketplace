# Restaurant Service API

Routes served under `/restaurants` via the Traefik gateway.

## Restaurants — `/restaurants`

| Method | Path | Auth | Description |
|---|---|---|---|
| `PATCH` | `/restaurants/{id}/address` | JWT | Update address |
| `PATCH` | `/restaurants/{id}/delivery` | JWT | Update delivery settings (pickup, delivery type, radius, fee, minimum order) |
