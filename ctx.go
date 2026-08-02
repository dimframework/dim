package dim

import (
	"encoding/json"
	"net/http"
)

type Ctx struct {
	w http.ResponseWriter
	r *http.Request
}

func Of(w http.ResponseWriter, r *http.Request) *Ctx {
	return &Ctx{w, r}
}

// Request
func (c *Ctx) Param(key string) string {
	return GetParam(c.r, key)
}

func (c *Ctx) Query(key string) string {
	return GetQueryParam(c.r, key)
}

func (c *Ctx) Queries(keys ...string) map[string]string {
	return GetQueryParams(c.r, keys...)
}

func (c *Ctx) Header(key string) string {
	return GetHeaderValue(c.r, key)
}

func (c *Ctx) Cookie(key string) string {
	return GetCookie(c.r, key)
}

func (c *Ctx) AuthToken() (string, bool) {
	return GetAuthToken(c.r)
}

func (c *Ctx) User() (Authenticatable, bool) {
	return GetUser(c.r)
}

func (c *Ctx) Claims() map[string]interface{} {
	return GetClaims(c.r)
}

func (c *Ctx) RequestID() string {
	return GetRequestID(c.r)
}

// ClientIP mengembalikan IP address klien.
// Hasilnya mengikuti middleware ClientIP bila terpasang, selain itu RemoteAddr.
func (c *Ctx) ClientIP() string {
	return GetClientIP(c.r)
}

// Request mengembalikan *http.Request yang dibungkus Ctx.
// Berguna untuk kebutuhan yang belum punya helper di Ctx, misalnya memanggil
// GetClientIPWithTrustedProxies secara langsung atau mengakses r.Context().
func (c *Ctx) Request() *http.Request {
	return c.r
}

// ResponseWriter mengembalikan http.ResponseWriter yang dibungkus Ctx.
func (c *Ctx) ResponseWriter() http.ResponseWriter {
	return c.w
}

func (c *Ctx) Bind(v interface{}) error {
	return json.NewDecoder(c.r.Body).Decode(v)
}

func (c *Ctx) Validate() *Validator {
	return NewValidator()
}

// Response
func (c *Ctx) JSON(status int, data interface{}) error {
	return Json(c.w, status, data)
}

func (c *Ctx) OK(data interface{}) error {
	return OK(c.w, data)
}

func (c *Ctx) Created(data interface{}) error {
	return Created(c.w, data)
}

func (c *Ctx) NoContent() error {
	return NoContent(c.w)
}

func (c *Ctx) BadRequest(message string, errors FieldErrors) error {
	return BadRequest(c.w, message, errors)
}

func (c *Ctx) Unauthorized(message string) error {
	return Unauthorized(c.w, message)
}

func (c *Ctx) Forbidden(message string) error {
	return Forbidden(c.w, message)
}

func (c *Ctx) NotFound(message string) error {
	return NotFound(c.w, message)
}

func (c *Ctx) Conflict(message string, errors FieldErrors) error {
	return Conflict(c.w, message, errors)
}

func (c *Ctx) InternalServerError(message string) error {
	return InternalServerError(c.w, message)
}

func (c *Ctx) TooManyRequests(retryAfterSeconds int) error {
	return TooManyRequests(c.w, retryAfterSeconds)
}

func (c *Ctx) MethodNotAllowed(message string) error {
	return MethodNotAllowed(c.w, message)
}

func (c *Ctx) Gone(message string) error {
	return Gone(c.w, message)
}

func (c *Ctx) UnprocessableEntity(message string, errors FieldErrors) error {
	return UnprocessableEntity(c.w, message, errors)
}

func (c *Ctx) ServiceUnavailable(message string) error {
	return ServiceUnavailable(c.w, message)
}

func (c *Ctx) AppError(appErr *AppError) error {
	return JsonAppError(c.w, appErr)
}
