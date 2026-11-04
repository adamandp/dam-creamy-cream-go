# Architecture and dependency rules

```text
cmd/api/main.go
       |
internal/app/bootstrap.go     <- only composition root creates infrastructure and services
       +-- infrastructure.NewPostgres   -> *gorm.DB (single pool)
       +-- infrastructure.NewRedis      -> *redis.Client (single pool)
       +-- infrastructure.NewMinIO -> *MinIOStore (private S3 bucket, public /media route)
       +-- sessions.NewService(db, accessSecret, refreshSecret)
       +-- roles.NewService(db)
       +-- users.NewService(db, sessions, minioStore)
       +-- auth.NewService(db, sessions)
       +-- addresses.NewService(db, sessions)
       +-- products.NewService(db, minioStore)
       +-- carts.NewService(db, sessions)
       +-- categories.NewService(db, redis)
       +-- inventory.NewService(db)
       +-- media.NewService(minioStore) -> GET /media/:scope/:id/:filename
       +-- health.NewService(sqlDB)
```

Go uses manual constructor injection. No cross-module NestJS-style `imports` or shared mutable package globals. Modules may rely on shared `internal/domain` models and interfaces or services injected by bootstrap, but do not construct other services themselves. One `gorm.DB` handles pooling; checkout will use a shared `Transaction(tx *gorm.DB)` context for all substeps when implemented. Avoid a module importing another module back through a cyclic chain.

Core source lives in `internal/`. The `playground` directory from the earlier discussion was a misunderstanding; it is **not** part of the delivered structure. Go test source lives in `tests/`.
