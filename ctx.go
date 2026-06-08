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

func (c *Ctx) ClientIP() string {
	return GetClientIP(c.r)
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

func (c *Ctx) BadRequest(message string, errors map[string]string) error {
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

func (c *Ctx) Conflict(message string, errors map[string]string) error {
	return Conflict(c.w, message, errors)
}

func (c *Ctx) InternalServerError(message string) error {
	return InternalServerError(c.w, message)
}

func (c *Ctx) TooManyRequests(retryAfterSeconds int) error {
	return TooManyRequests(c.w, retryAfterSeconds)
}

func (c *Ctx) AppError(appErr *AppError) error {
	return JsonAppError(c.w, appErr)
}
