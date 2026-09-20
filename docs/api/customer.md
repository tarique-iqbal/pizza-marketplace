# Customer Service API

Everything is served through the Traefik gateway at `/customers`, protected by the `jwt@docker` forward-auth middleware. Identity comes from the `X-User-ID`/`X-User-Role` headers Traefik injects after identity-service validates the JWT; this service does not parse tokens itself. Any authenticated user can call these routes (no role gate), and every route is scoped to the caller's own data.

## Profile — `/customers/me`

| Method | Path | Auth | Notes |
|---|---|---|---|
| `GET` | `/customers/me` | authenticated user | The caller's profile |
| `PATCH` | `/customers/me/phone` | authenticated user | Set the caller's phone number |

`GET /customers/me` returns `{id, email, firstName, lastName, phone?, updatedAt?}`. Name and email come from identity-service and are read-only here; `phone` and `updatedAt` are omitted until a phone has been set.

`PATCH /customers/me/phone` takes `{phone}`, required, at most 32 characters (`422` otherwise), and returns the updated profile. The change is published as `customer.phone_updated` in the same database transaction. Order-service uses that phone as the delivery contact number at checkout.

A customer's profile row is created when identity-service's `user.registered` event is consumed, which is asynchronous. A `GET` or `PATCH` sent before that has landed returns `404 Not Found`.

## Saved addresses — `/customers/me/addresses`

| Method | Path | Auth | Notes |
|---|---|---|---|
| `GET` | `/customers/me/addresses` | authenticated user | The caller's saved addresses, default first |
| `DELETE` | `/customers/me/addresses/:id` | authenticated user | Delete one address |
| `POST` | `/customers/me/addresses/:id/default` | authenticated user | Make one address the default |

There is no `POST` to create an address. An address is saved when a customer checks out with `saveAddress: true` on a delivery order: order-service publishes `order.address_saved` and this service creates the address from it. Saving the same house, street, city and postal code again does nothing, and the customer's first saved address becomes their default.

`GET` returns a bare JSON array (no pagination, since a customer has a handful of addresses), ordered default first and then oldest first:

```json
[
  {
    "id": "01a0bb6a-2fec-79aa-a76e-8c3fca232220",
    "house": "20",
    "street": "Beta St",
    "city": "Berlin",
    "postalCode": "10117",
    "isDefault": true,
    "createdAt": "2026-09-19T20:44:53.612765Z"
  }
]
```

The field names match order-service's checkout `deliveryAddress` (`house`, `street`, `postalCode`, `city`), so a saved address can be copied into a checkout request as it is.

`DELETE` returns `204 No Content`. Deleting the default address does not promote another one; the customer picks a new default explicitly.

`POST .../default` returns the updated address. The previous default is cleared in the same transaction, so a customer never has two.

`:id` must be a UUID (`400 Bad Request` otherwise). An address that doesn't exist and one that belongs to someone else both return `404 Not Found`, so the two cases can't be told apart from outside.

## Errors

| Status | When |
|---|---|
| `400` | `:id` is not a valid UUID |
| `401` | Missing or invalid credentials (rejected by the gateway before reaching this service) |
| `404` | Profile row not created yet, or address missing or not the caller's |
| `422` | `phone` missing or longer than 32 characters |
