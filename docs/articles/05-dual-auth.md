# JWT Meets API Keys: Building Dual-Auth in Go Without the Usual Mistakes

*Part 5 of the "Building Production-Ready AI Agent APIs in Go" series*

---

Most authentication tutorials make you choose: JWT tokens or API keys. User-facing web apps usually want JWTs. Machine-to-machine integrations usually want long-lived API keys. If your product serves both — humans using a web app and developers integrating via code — you need both.

This project supports both in a single authentication pipeline. The decision point is one line: does the token start with `"sk-"`? If yes, it is an API key. Otherwise, it is a JWT. Everything downstream — rate limiting, tool permissions, user lookup — is identical.

Let us walk through how it is built.

---

## The Two Authentication Flows

Before code, the high-level picture:

**JWT Flow (for web apps):**
```
User logs in → POST /v1/auth/login
→ bcrypt verify password
→ create Token record in DB
→ generate JWT (access token, 15 min) + JWT (refresh token, 7 days)
→ return both tokens to client
→ client sends: Authorization: Bearer eyJhbGci...
```

**API Key Flow (for developers):**
```
User creates key → POST /v1/auth/tokens (authenticated)
→ generate "sk-{uuid}" raw key
→ SHA-256 hash the raw key
→ store Token record with hash (NEVER store raw)
→ return raw key once (user saves it)
→ developer sends: Authorization: Bearer sk-abc123...
```

Both result in a `Token` entity and a `User` entity being available in the request context. The rest of the codebase uses them identically.

---

## The JWT Manager: Generating and Validating Tokens

The JWT utility lives in `internal/pkg/jwt/jwt.go`:

```go
type Claims struct {
    UserID    uuid.UUID `json:"uid"`
    TokenID   uuid.UUID `json:"tid"`  // References the Token record in DB
    TokenType string    `json:"typ"`  // "access" or "refresh"
    jwt.RegisteredClaims
}

type JWTManager struct {
    secret          []byte
    accessTokenTTL  time.Duration   // 15 minutes
    refreshTokenTTL time.Duration   // 7 days
    issuer          string
}
```

Why does the JWT carry a `TokenID` (UUID) in addition to `UserID`?

Because JWTs are stateless by design. Once issued, you cannot invalidate a JWT before its expiry time — unless you maintain a blocklist. Instead of a blocklist, we take a different approach: every JWT references a `Token` record in the database. When we validate the JWT, we load the `Token` record and check `token.IsValid()`. If the token has been revoked, `IsValid()` returns false, and the JWT is effectively invalidated.

This gives us the best of both worlds: JWTs are fast (no DB lookup for validity in theory) but can be revoked by updating the DB record.

```go
func (m *JWTManager) GenerateAccessToken(userID, tokenID uuid.UUID) (string, error) {
    claims := Claims{
        UserID:    userID,
        TokenID:   tokenID,
        TokenType: "access",
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.accessTokenTTL)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    m.issuer,
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(m.secret)
}
```

The validation is standard HMAC-SHA256:

```go
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, errors.New("unexpected signing method")
        }
        return m.secret, nil
    })
    if err != nil {
        return nil, err
    }
    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, errors.New("invalid token")
    }
    return claims, nil
}
```

The signing method check (`if _, ok := token.Method.(*jwt.SigningMethodHMAC)`) is important — it prevents the "algorithm confusion" attack where an attacker crafts a JWT with `alg: none` or `alg: RS256` and your server accidentally accepts it. Always verify the algorithm explicitly.

---

## Hashing API Keys: Why You Never Store the Raw Value

```go
// HashToken hashes a raw token for storage.
func HashToken(rawToken string) string {
    hash := sha256.Sum256([]byte(rawToken))
    return hex.EncodeToString(hash[:])
}

// GenerateAPIKey generates a random API key.
func GenerateAPIKey() string {
    return "sk-" + uuid.New().String()
}
```

When a user creates an API key, `GenerateAPIKey()` produces something like `sk-550e8400-e29b-41d4-a716-446655440000`. We:
1. Return the raw key to the user — this is the **only time they will see it**
2. Hash it with SHA-256
3. Store only the hash in the database

This mirrors how password hashing works (except we use SHA-256 instead of bcrypt, because API keys are already high-entropy random values). If someone breaches your database, they get only hashes — worthless without the raw keys.

When the user later sends the raw key, we hash it again and look it up:

```go
func (uc *ValidateTokenUseCase) validateAPIKey(ctx context.Context, rawKey string) (*ValidateTokenResult, error) {
    tokenHash := jwt.HashToken(rawKey)
    token, err := uc.tokenRepo.FindByHash(ctx, tokenHash)
    // ...
}
```

The `sk-` prefix is important. It makes API keys visually distinct from JWTs, enables tooling to detect accidental exposure (GitHub's secret scanning looks for `sk-` patterns), and most importantly, it enables the single-line routing in the `ValidateTokenUseCase`.

---

## The ValidateTokenUseCase: One Branching Point

The entire dual-auth routing logic lives in `internal/application/usecase/auth/validate_token.go`:

```go
func (uc *ValidateTokenUseCase) Execute(ctx context.Context, rawToken string) (*ValidateTokenResult, error) {
    // Check if it's an API key (starts with "sk-")
    if strings.HasPrefix(rawToken, "sk-") {
        return uc.validateAPIKey(ctx, rawToken)
    }
    // Otherwise treat as JWT
    return uc.validateJWT(ctx, rawToken)
}
```

One line. That is the entirety of the routing decision. The API key path goes to `validateAPIKey()`, the JWT path goes to `validateJWT()`. Both return `*ValidateTokenResult` with the same shape.

The API key path:

```go
func (uc *ValidateTokenUseCase) validateAPIKey(ctx context.Context, rawKey string) (*ValidateTokenResult, error) {
    tokenHash := jwt.HashToken(rawKey)

    token, err := uc.tokenRepo.FindByHash(ctx, tokenHash)
    if err != nil {
        return nil, fmt.Errorf("invalid API key")  // Don't leak "not found" vs "wrong key"
    }

    if !token.IsValid() {
        return nil, fmt.Errorf("API key is expired or revoked")
    }

    user, err := uc.userRepo.FindByID(ctx, token.UserID)
    if err != nil {
        return nil, fmt.Errorf("user not found")
    }

    if !user.IsActive {
        return nil, fmt.Errorf("user account is inactive")
    }

    // Update last used (fire and forget — don't add latency to every request)
    go uc.tokenRepo.UpdateLastUsed(context.Background(), token.ID)

    return &ValidateTokenResult{User: user, Token: token}, nil
}
```

Notice the error message for a missing token: `"invalid API key"` instead of `"token not found"`. This prevents enumeration attacks — an attacker cannot distinguish between "wrong key" and "nonexistent key" from the error response.

The JWT path:

```go
func (uc *ValidateTokenUseCase) validateJWT(ctx context.Context, tokenString string) (*ValidateTokenResult, error) {
    claims, err := uc.jwtMgr.ValidateToken(tokenString)
    if err != nil {
        return nil, fmt.Errorf("invalid token: %w", err)
    }

    // Load the Token record from DB to check IsValid() (revocation check)
    token, err := uc.tokenRepo.FindByID(ctx, claims.TokenID)
    if err != nil {
        return nil, fmt.Errorf("token not found")
    }

    if !token.IsValid() {
        return nil, fmt.Errorf("token is expired or revoked")
    }

    user, err := uc.userRepo.FindByID(ctx, claims.UserID)
    if err != nil {
        return nil, fmt.Errorf("user not found")
    }

    if !user.IsActive {
        return nil, fmt.Errorf("user account is inactive")
    }

    go uc.tokenRepo.UpdateLastUsed(context.Background(), token.ID)

    return &ValidateTokenResult{User: user, Token: token}, nil
}
```

The JWT path does one extra DB lookup compared to the API key path: it loads `Token` by the `TokenID` from claims. This is the revocation check — even if the JWT's expiry time has not passed, if the DB record is revoked, `token.IsValid()` returns false.

---

## The Fire-and-Forget UpdateLastUsed Pattern

Both validation paths end with:

```go
go uc.tokenRepo.UpdateLastUsed(context.Background(), token.ID)
```

This goroutine fires and is never awaited. It updates the `last_used_at` timestamp on the token in the database — useful for identifying stale tokens that should be cleaned up, and for audit trails.

Why fire-and-forget? Because adding a synchronous database write to every authenticated request would increase latency for every request. The `last_used_at` update is best-effort audit data. If the goroutine fails occasionally (DB hiccup, server shutdown), that is acceptable — the request still succeeded, and the timestamp is only approximate anyway.

Note the `context.Background()` instead of passing the request context. If we passed the request context, the goroutine might be cancelled when the request completes, before the DB write finishes. `context.Background()` ensures the goroutine has a fresh, uncancelled context.

---

## The Auth Middleware: Context Injection

The auth middleware in `internal/infrastructure/http/middleware/auth.go` wraps the `ValidateTokenUseCase` and injects results into the request context:

```go
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authorization header")
            return
        }

        // Strip "Bearer " prefix if present
        token := strings.TrimPrefix(authHeader, "Bearer ")
        token = strings.TrimSpace(token)

        result, err := m.validateToken.Execute(r.Context(), token)
        if err != nil {
            response.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", err.Error())
            return
        }

        ctx := context.WithValue(r.Context(), UserContextKey, result.User)
        ctx = context.WithValue(ctx, TokenContextKey, result.Token)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

After the middleware runs, every handler downstream can call:

```go
user := middleware.GetUserFromContext(r.Context())   // *entity.User
token := middleware.GetTokenFromContext(r.Context()) // *entity.Token
```

The helper functions use typed context keys to avoid the `interface{}` collision risk:

```go
type contextKey string

const (
    UserContextKey  contextKey = "user"
    TokenContextKey contextKey = "token"
)

func GetUserFromContext(ctx context.Context) *entity.User {
    user, ok := ctx.Value(UserContextKey).(*entity.User)
    if !ok {
        return nil
    }
    return user
}
```

Using a custom type (`contextKey string`) for context keys prevents key collisions with other middleware or libraries that might use `"user"` as a plain string key. The type assertion `ctx.Value(UserContextKey).(*entity.User)` returns `nil` if the value was stored with a different key type or is a different concrete type.

---

## How Per-Token ACLs Flow Through the System

Once the token is in context, it is used downstream to filter what the user can do:

**In the tool handler**, available tools are filtered by the token:

```go
func (h *ToolHandler) ListTools(w http.ResponseWriter, r *http.Request) {
    token := middleware.GetTokenFromContext(r.Context())

    var tools []toolspec.Tool
    if token != nil {
        tools = h.registry.ListForToken(token.AllowedTools)
    } else {
        tools = h.registry.List()
    }
    response.JSON(w, http.StatusOK, map[string]any{"tools": tools})
}
```

**In the rate limiter middleware**, per-token rate limits are enforced:

```go
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := GetTokenFromContext(r.Context())
        if token == nil {
            next.ServeHTTP(w, r)
            return
        }

        minuteKey := fmt.Sprintf("ratelimit:%s:minute:%d", token.ID, time.Now().Unix()/60)
        count, _ := rl.redis.Incr(ctx, minuteKey).Result()

        if count > int64(token.RateLimitPerMinute) {
            response.Error(w, http.StatusTooManyRequests, "RATE_LIMIT", "Rate limit exceeded")
            return
        }
        // ...
    })
}
```

The rate limit key includes `token.ID`, so each token has its own counter. A token with `RateLimitPerMinute: 1000` and a token with `RateLimitPerMinute: 10` are tracked separately, even for the same user account.

---

## The Login Use Case: Where Tokens Are Created

When a user logs in (`POST /v1/auth/login`), the `LoginUseCase` creates a `Token` record and issues a JWT:

```go
func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
    user, err := uc.userRepo.FindByEmail(ctx, input.Email)
    if err != nil {
        return nil, apperrors.ErrInvalidCredentials
    }

    // bcrypt comparison (constant-time, prevents timing attacks)
    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
        return nil, apperrors.ErrInvalidCredentials
    }

    // Create a Token record in the DB (this is what the JWT's TokenID references)
    tokenID := uuid.New()
    accessToken, _ := uc.jwtMgr.GenerateAccessToken(user.ID, tokenID)
    refreshToken, _ := uc.jwtMgr.GenerateRefreshToken(user.ID, tokenID)

    token := entity.NewAccessToken(user.ID, jwt.HashToken(accessToken), "session")
    token.ID = tokenID  // Use the same ID that's baked into the JWT
    uc.tokenRepo.Create(ctx, token)

    return &LoginOutput{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        User:         user,
    }, nil
}
```

The `TokenID` is generated once and embedded in both the JWT claims and the DB record. This is the link that enables revocation: when you call `ValidateJWT`, the `TokenID` from claims lets us load the DB record and check `IsValid()`.

---

## What We Just Learned

- Dual auth routes on a single `strings.HasPrefix(rawToken, "sk-")` check in `ValidateTokenUseCase`
- JWTs embed `TokenID` to enable revocation via a DB `IsValid()` check
- API keys are SHA-256 hashed before storage — the raw value is shown once and never stored
- The `sk-` prefix distinguishes API keys from JWTs, prevents enumeration, and enables secret scanning
- `go uc.tokenRepo.UpdateLastUsed(context.Background(), token.ID)` is a fire-and-forget pattern — best-effort audit without adding latency
- Auth middleware injects `*entity.User` and `*entity.Token` into context for downstream use
- Rate limits and tool permissions are per-token, tracked in Redis by `token.ID`

---

*Next in the series: [Article 6 — Server-Sent Events in Go: Building Real-Time AI Streaming](#)*

*Previous: [Article 4 — Eino: ByteDance's LangGraph for Go](#)*
