# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

