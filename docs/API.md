# Cryptocash TON Battery — API referenca

Bazna putanja: `/` · Swagger UI: `/swagger/index.html`

Autentikacija (dva „Authorize" dugmeta u Swaggeru):
- **BearerAuth** — `Authorization: Bearer <JWT>` (JWT iz `/ton-proof/check`)
- **AdminToken** — `X-Admin-Token: <ADMIN_TOKEN>`

Svi odgovori su `application/json`. Iznosi: `*Nano` = nanotoni (1 TON = 1_000_000_000), `charge`/`balance` = bodovi (cijeli brojevi).

---

## Javne rute (bez auth)

### GET /health
```json
{ "status": "OK", "message": "TON Battery API works!" }
```

### GET /firebase/status
200:
```json
{ "connected": true, "message": "Firestore connection OK" }
```
503:
```json
{ "connected": false, "message": "Firestore connection failed", "error": "..." }
```

### GET /products
```json
{ "products": [
  { "id": "charges_100",  "charges": 100,  "price": "€4.99"  },
  { "id": "charges_500",  "charges": 500,  "price": "€19.99" },
  { "id": "charges_1200", "charges": 1200, "price": "€39.99" }
] }
```

### GET /price
```json
{ "nanoPerCharge": 4000000, "tonPrice": 2.5, "currency": "usd", "centsPerCharge": 1, "fallback": false }
```
`fallback: true` znači da kurs nije dostupan pa se koristi statični `POLICY_NANO_PER_CHARGE`.

### GET /relayer/status
Online:
```json
{ "address": "EQ...", "online": true, "balanceNano": 11999999997, "seqno": 1, "deployed": false }
```
Mreža nedostupna:
```json
{ "address": "EQ...", "online": false, "error": "chain-unreachable" }
```
503: `{ "error": "relayer-not-configured" }`

---

## Auth (ton_proof → JWT)

### GET /ton-proof/payload
```json
{ "payload": "<hex nonce>" }
```
500: `{ "error": "nonce-issue-failed" }`

### POST /ton-proof/check
Request:
```json
{
  "address": "0:abc...",
  "network": "-3",
  "public_key": "<hex 64>",
  "proof": {
    "timestamp": 1690000000,
    "domain": { "length_bytes": 9, "value": "localhost" },
    "signature": "<base64>",
    "payload": "<hex nonce>"
  },
  "wallet_state_init": "<base64 opcionalno>"
}
```
200:
```json
{ "token": "<JWT>", "expiresAt": 1690086400, "userId": "<uuid>", "publicKey": "<hex>", "address": "EQ...", "newUser": true }
```
401 (reject reason u `error`): `invalid-nonce`, `bad-public-key`, `bad-address`, `wrong-network`, `not-a-v5-wallet`, `domain-not-allowed`, `stale-proof`, `bad-signature`
400: `{ "error": "malformed-request" }`
500: `{ "error": "jwt-not-configured" }`

---

## Zaštićene rute (BearerAuth)

### GET /auth/me
```json
{ "userId": "<uuid>", "publicKey": "<hex>" }
```

### GET /balance
```json
{ "userId": "<uuid>", "balance": 100, "reserved": 10, "available": 90 }
```

### GET /transactions
```json
{ "userId": "<uuid>", "transactions": [
  { "id": "<uuid>", "op": "credit", "amount": 100, "balanceAfter": 100, "reservedAfter": 0, "reason": "test", "createdAt": "2026-10-08T10:00:00Z" }
] }
```
`op`: `credit` | `reserve` | `settle` | `release`

### POST /wallet/emulate
Request: `{ "boc": "<base64>", "ignoreSignature": true }`
200:
```json
{ "supported": true, "allowed": true, "rejectReason": "", "charge": 7 }
```
Response header-i: `Supported-By-Battery`, `Allowed-By-Battery`, `Reject-Reason`.
`rejectReason` može biti: `malformed-message`, `message-too-large`, `unsupported-operation`, `too-many-messages`, `destination-blocked`, `ttl-too-short`, `emulation-failed`, `fee-too-high`, `insufficient-charges`, `rate-limited`.

### POST /message
Request: `{ "boc": "<base64>", "ignoreSignature": false }`
Uspjeh:
```json
{ "status": "confirmed", "txHash": "<hex>", "charge": 7 }
```
Odbijeno (nije poslato):
```json
{ "allowed": false, "supported": true, "rejectReason": "insufficient-charges", "charge": 7 }
```
400: `{ "error": "missing-boc" | "malformed-request" }`
502: `{ "error": "send-failed", "detail": "..." }`
503: `{ "error": "relayer-sender-not-ready" }`
Isti decision header-i kao `/wallet/emulate`. Kad je kill switch upaljen ili limit pređen → `rejectReason`: `battery-paused` | `rate-limited` | `velocity-exceeded`.

### POST /print/quote
Request: `{ "toAddress": "0Q..." }`
```json
{ "quoteId": "<uuid>", "bufferTON": "0.075", "charge": 19, "expiresAt": 1690000300 }
```

### POST /print/execute
Request: `{ "quoteId": "<uuid>" }`
Uspjeh:
```json
{ "quoteId": "<uuid>", "status": "completed", "txHash": "<hex>", "charge": 19 }
```
Greške: `{ "error": "print-job-not-found" | "already-processed" | "quote-expired" | "insufficient-charges" }`, 502 `{ "error": "buffer-send-failed", "detail": "..." }`

### GET /print/{id}
```json
{ "quoteId": "<uuid>", "status": "completed", "toAddress": "0Q...", "bufferTON": "0.075", "charge": 19, "txHash": "<hex>" }
```
`status`: `quoted` | `buffer_pending` | `completed` | `failed` | `expired`

---

## Admin rute (AdminToken)

### POST /admin/users/{id}/credit
Request: `{ "amount": 100, "idempotencyKey": "k1", "reason": "test" }`
```json
{ "userId": "<id>", "balance": 100, "reserved": 0, "available": 100, "idempotent": false, "entryId": "<uuid>" }
```
`idempotent: true` = isti `idempotencyKey` je već korišten, balans se ne mijenja.

### GET /admin/users/{id}
```json
{ "userId": "<id>", "balance": 100, "reserved": 0, "available": 100 }
```

### POST /admin/killswitch
Request: `{ "paused": true }` → `{ "paused": true }`

### GET /admin/killswitch
```json
{ "paused": false }
```

### GET /admin/users/{id}/reconcile
```json
{ "userId": "<id>", "accountBalance": 100, "accountReserved": 0, "computedBalance": 100, "computedReserved": 0, "consistent": true }
```
`consistent: false` = account doc se razišao s ledger zapisima (treba istraga).

---

## Dev rute (samo `DEV_SIGN=true`) — brišu se prije produkcije
Svaka nosi header `X-Dev-Route: TO-BE-DELETED`.

### GET /ton-proof/dev-sign?payload=<hex>&domain=localhost
Vraća gotov body za `/ton-proof/check` (potpisan test-walletom): `{ address, network, public_key, proof{...} }`

### POST /relayer/dev-send?to=<addr>&amount=0.05&comment=...
```json
{ "txHash": "<hex>", "from": "EQ...", "to": "0Q...", "amount": "0.05", "explorer": "https://testnet.tonviewer.com/EQ..." }
```

### POST /emulate
Request: `{ "boc": "<base64>", "ignoreSignature": true }`
```json
{ "success": true, "totalFeesNano": 2500000, "outMessages": 1, "destinations": ["0:..."] }
```
