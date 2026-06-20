# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

---

## [v0.7.2] - 2026-06-20

### Added
- **`PostgresDatabase.WritePool()` dan `ReadPools()` accessors**: Mengekspos `*pgxpool.Pool` yang mendasari `PostgresDatabase` untuk integrasi lanjutan — memungkinkan penggunaan pustaka yang membutuhkan akses `pgx` langsung seperti job queue (`riverqueue/river`) dan `LISTEN/NOTIFY` broker. Konsisten dengan pola escape-hatch `PostgresTx.PgxTx()` yang sudah ada. Interface `Database` tidak berubah; accessor hanya tersedia di tipe konkret `*PostgresDatabase` dan diakses via type assertion. Closes [#8](https://github.com/dimframework/dim/issues/8).

---

## [v0.7.1] - 2026-06-11

### Changed
- **`Validator.ErrorMap()` return type**: Changed from `map[string]string` to `FieldErrors` (type alias for `map[string]any`). This allows seamless integration with `BadRequest()` and `JsonError()` — no adapter function needed. All signatures updated; `FieldErrorsFrom()` is now redundant and can be removed in application code.
  - **Before**: `dim.BadRequest(w, msg, dim.FieldErrorsFrom(v.ErrorMap()))`
  - **After**: `dim.BadRequest(w, msg, v.ErrorMap())`
  - ⚠️ **Breaking Change**: Code that type-asserts `v.ErrorMap()` as `map[string]string` will panic. Type-assert as `map[string]any` instead, or use the typed accessor methods `v.Error(field string)` and `v.Errors(field string) []string`.

### Added
- **`Validator.WithFullErrors()` method**: Enables accumulating all errors per field instead of first-error-wins. Can be chained at start, middle, or end of the validation chain. After `WithFullErrors()` is called, subsequent rules collect all errors in `map[string][]string` internally and merge them into `FieldErrors` on `ErrorMap()`.
  - **Example**: `v := dim.NewValidator().WithFullErrors().Required(...).Email(...)`
  - Errors are returned as `FieldErrors{"field": []string{"error1", "error2"}}`

### Updated
- **Docs**: Updated `13-validation.md`, `14-error-handling.md`, and `07-response-helpers.md` to reflect the new `ErrorMap()` behavior, `WithFullErrors()` patterns, and removed `FieldErrorsFrom()` adapter usage.

---

## [v0.7.0] - 2026-06-08

### Added
- **`Ctx` helper (`dim.Of`)**: Added opt-in ergonomic wrapper that bundles `http.ResponseWriter` and `*http.Request` into a single `*Ctx` object. Reduces boilerplate in handlers that call many helpers — use `c := dim.Of(w, r)` and replace `dim.GetParam(r, "id")` with `c.Param("id")`, `dim.OK(w, data)` with `c.OK(data)`, etc. No breaking changes; existing handlers continue to work unchanged. Closes [#6](https://github.com/dimframework/dim/issues/6).
  - Request helpers: `Param`, `Query`, `Queries`, `Header`, `Cookie`, `AuthToken`, `User`, `Claims`, `RequestID`, `ClientIP`
  - `Bind(&v)` — decodes JSON request body into a struct
  - `Validate()` — shorthand for `dim.NewValidator()`
  - Response helpers: `JSON`, `OK`, `Created`, `NoContent`, `BadRequest`, `Unauthorized`, `Forbidden`, `NotFound`, `Conflict`, `InternalServerError`, `TooManyRequests`, `AppError`
- **`Ctx` docs**: Added "Ctx Helper — Ergonomic Syntax" section to `docs/07-response-helpers.md` with method tables, side-by-side comparison, and a complete `CreateUser` handler example.

---

## [v0.6.2] - 2026-06-04

### Changed
- **Pure Go SQLite driver**: Replaced `github.com/mattn/go-sqlite3` (CGO) with `modernc.org/sqlite` (pure Go). No CGO toolchain required — builds work out of the box on all platforms without a C compiler. API and behavior are unchanged.

---

## [v0.6.1] - 2026-05-30

### Added
- **`WithClaimsProvider` tests**: Added comprehensive tests for `WithClaimsProvider` to verify custom claims are correctly embedded in access tokens.
- **Authentication & Token API docs**: Added full API reference section for `AuthService`, `JWTManager`, `BrancaManager`, `ClaimsProvider`, and `Authenticatable` in `docs/23-api-reference.md`.
- **`WithClaimsProvider` docs**: Added usage guide for `WithClaimsProvider` in `docs/12-authentication.md`.

### Fixed
- **`BRANCA_KEY` not validated at startup**: `Validate()` now decodes `BRANCA_KEY` at startup and returns a descriptive error if the key is invalid (wrong length or format). Previously, an invalid key was only caught at runtime when `NewBrancaManager` was called.
- **`JWT_SECRET` always required even when using Branca**: `Validate()` no longer requires `JWT_SECRET` when `BRANCA_KEY` is set. If `BRANCA_KEY` is present, Branca is treated as the active token provider and JWT validation is skipped entirely.

---

## [v0.6.0] - 2026-05-07

### Added
- **Branca Token Provider**: Added `BrancaManager` as an alternative to `JWTManager`. Branca tokens encrypt the payload using XChaCha20-Poly1305 — claims are unreadable by the client, suitable for sensitive payloads or internal services.
- **`TokenManager` Interface**: Introduced `TokenManager` interface and `TokenClaims` type alias to abstract token operations. Both `JWTManager` and `BrancaManager` implement this interface, enabling provider switching without changing application code.
- **`NewAuthServiceWithManager`**: New constructor for `AuthService` that accepts any `TokenManager` implementation, enabling Branca (or future providers) to be used with `AuthService`.
- **`BrancaConfig`**: New config struct with `BRANCA_KEY`, `BRANCA_ACCESS_TOKEN_EXPIRY`, and `BRANCA_REFRESH_TOKEN_EXPIRY` environment variables.
- **Base64 PEM support for JWT keys**: `JWT_PRIVATE_KEY` and `JWT_PUBLIC_KEYS` now accept base64-encoded PEM content in addition to file paths and raw PEM strings — recommended for Docker/Kubernetes environments where newlines in env vars are problematic.
- **Hybrid static map + radix tree router**: Replaced `http.ServeMux` routing backend with a two-tier dispatch. Static routes (no URL parameters) are stored in an O(1) map; dynamic routes (`{param}`, `{path...}`) use a chi-style radix tree (O(k) per path segment). `http.ServeMux` is retained as a fallback for `Static()` and `SPA()` file serving only.
- **Migration database connection**: Added `NewMigrationDatabase` constructor and `DB_MIGRATION_HOST/PORT/USERNAME/PASSWORD` env vars for a dedicated migration database connection. Falls back to the Write connection when any field is unset. `WithMigrationDB` adds it to the CLI `Console`; `CommandContext.MigrationDB` exposes it inside migration commands.

### Changed
- **`RequireAuth` and `OptionalAuth`**: Parameter type changed from `*JWTManager` to `TokenManager` interface. Fully backward compatible — existing code passing `*JWTManager` continues to work unchanged.
- **`JWTManager.VerifyToken`**: Return type changed from `jwt.MapClaims` to `TokenClaims` (a type alias for `map[string]interface{}`). Fully backward compatible for map access patterns.

### Fixed
- **Branca base62 leading zeros**: `brancaBase62Encode`/`brancaBase62Decode` now preserve leading zero bytes in the token binary, preventing silent payload corruption when the header starts with `0x00`.
- **`decodeBrancaKey` ambiguous format detection**: Length guards ensure a 32-character raw key is never misinterpreted as base64, and base64 variants are only attempted at their canonical lengths (44 for std, 43 for raw-URL).
- **Branca reserved claim protection**: `GenerateAccessToken` now returns an error if `extraClaims` contains a reserved key (`sub`, `sid`, `jti`, `email`, `iat`, `exp`, `nbf`, `typ`), preventing silent overwrite of internal claims.

---

## [v0.5.0] - 2026-02-10

### Added
- **Auth Middleware Flexibility**: Added Functional Options pattern to `RequireAuth` middleware (`WithBearerToken`, `WithCookieToken`) allowing token extraction from Headers or Cookies.
- **CORS Support for Exposed Headers**: Added `ExposedHeaders` to `CORSConfig` and support for `CORS_EXPOSED_HEADERS` environment variable.
- **CSRF Token Expiration**: Added `CookieMaxAge` to `CSRFConfig` and `CSRF_COOKIE_MAX_AGE` environment variable (default: 12 hours) to allow CSRF cookies to expire.
- **CORS Vary Header**: Added `Vary: Origin` header to CORS responses to prevent cache poisoning.

### Changed
- **CSRF Error Code**: Changed CSRF validation failure status code from `403 Forbidden` to `419 Authentication Timeout` (Custom Status) to better distinguish CSRF issues from permission issues.
- **CORS Preflight Status**: Changed CORS preflight response status from `200 OK` to `204 No Content`.
- **CORS Logic**: Updated CORS middleware to pass through non-CORS `OPTIONS` requests (requests without `Origin` header) instead of swallowing them.
- **CORS Max-Age**: Fixed bug in `Access-Control-Max-Age` header where integer value was incorrectly converted to string.
- **Documentation**: Updated `middleware.md`, `configuration.md`, and `security.md` with new configuration options and best practices.
