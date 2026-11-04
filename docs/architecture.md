# Application architecture

Creamy Cream Go uses a modular monolith with explicit constructor injection.

```text
cmd/api/main.go
   |
internal/app/bootstrap.go
   |-- infrastructure.NewPostgres  -> shared GORM DB pool
   |-- infrastructure.NewRedis     -> shared cache client
   |-- infrastructure.NewMinIO     -> private image store
   |-- sessions.NewService(db, JWT secrets)
   |-- roles / users / auth / addresses
   |-- categories / products / carts / inventory
   |-- discounts / productdiscounts / vouchers / uservouchers
   |-- media / health
```

Each feature defines its own HTTP routes and services. The composition root constructs dependencies once, shares connections, and handles shutdown. GORM models use explicit table and column names from the application schema. Tests live under tests/, outside application source.

Feature packages should not create independent DB/Redis/MinIO connections. Use transactions for multi-step writes. The application does not automatically change schema definitions on startup.
