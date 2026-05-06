# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

