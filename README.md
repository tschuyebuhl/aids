# aids

Small helpers for building Go HTTP APIs: request logging, SPA fallbacks, Keycloak auth, query param parsing, and a few data/query utilities.

## Install

```sh
go get github.com/tschuyebuhl/aids
```

## HTTP middleware

Query params parsing with filters, sorting, and pagination:

```go
mux := http.NewServeMux()
handler := middleware.QueryParams(mux)

func list(w http.ResponseWriter, r *http.Request) {
    params := middleware.QueryParamsFromContext(r.Context())
    // params.Filter, params.Sort, params.Pagination
    _ = params
}
```

Query params URL example:

```
GET /api/habits?habit_id_exact="f86b053f-94ce-4c6f-b13a-e1208979218a"
```

Keycloak auth as middleware:

```go
auth := middleware.NewKeycloak(provider)
secured := auth.Handler(handler)
// or: httpx.Chain(handler, auth.Middleware())
```

Custom token mapping with extra JWT claims:

```go
type userEmailKey struct{}
type userRolesKey struct{}

var UserEmailKey = userEmailKey{}
var UserRolesKey = userRolesKey{}

auth := middleware.NewKeycloak(provider, middleware.WithTokenMapper(
    func(ctx context.Context, token *oidc.IDToken) (context.Context, error) {
        var claims struct {
            Email       string `json:"email"`
            RealmAccess struct {
                Roles []string `json:"roles"`
            } `json:"realm_access"`
        }

        if err := token.Claims(&claims); err != nil {
            return ctx, err
        }

        ctx = userctx.WithUserID(ctx, token.Subject)
        ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
        ctx = context.WithValue(ctx, UserRolesKey, claims.RealmAccess.Roles)
        return ctx, nil
    },
))
```

Per-route middleware:

```go
routes := []httpx.Route{
    {Pattern: "GET /health", Handler: health},
    {Pattern: "GET /health/secure", Handler: health, Use: []httpx.Middleware{auth.Middleware()}},
}
```

Route groups:

```go
api := httpx.Use(habits, auth.Middleware())
httpx.Register(mux, api)
```

## HTTP helpers

Request logging with panic recovery:

```go
logged := httpx.NewLogger(handler)
```

SPA fallback for embedded or static file servers:

```go
fs := http.FS(embedded)
fileServer := http.FileServer(fs)
serveIndex := httpx.ServeFileContents("index.html", fs)

mux.Handle("/", httpx.Intercept404(fileServer, serveIndex))
```

## Data/query helpers

Slugify and query helpers:

```go
slug := data.Slugify("Daily Focus")
_ = slug
```

Apply user scoping in bob queries:

```go
mods := []bob.Mod[*dialect.SelectQuery]{
    query.UserIDModifier(ctx),
}
```

## Full server wiring example

```go
var (
    //go:embed frontend/dist
    embeddedFS embed.FS
)

func routes(devMode bool, provider *oidc.Provider) http.Handler {
    mux := http.NewServeMux()

    if devMode {
        proxyURL, _ := url.Parse("http://localhost:5173")
        mux.Handle("/", httputil.NewSingleHostReverseProxy(proxyURL))
    } else {
        frontend, _ := fs.Sub(embeddedFS, "frontend/dist")
        httpFS := http.FS(frontend)
        fileServer := http.FileServer(httpFS)
        serveIndex := httpx.ServeFileContents("index.html", httpFS)
        mux.Handle("/", httpx.Intercept404(fileServer, serveIndex))
    }

    health := httpx.Routes(httpx.Route{
        Pattern: "GET /health",
        Handler: func(w http.ResponseWriter, r *http.Request) {
            w.WriteHeader(http.StatusOK)
        },
    })
    habits := &HabitRouter{} // implements httpx.Routable

    auth := middleware.NewKeycloak(provider)
    api := httpx.Use(habits, auth.Middleware())

    httpx.Register(mux, api, health)

    return httpx.Chain(
        mux,
        middleware.QueryParams,
        httpx.LoggerMiddleware(),
    )
}
```

## Tests

```sh
go test ./...
```

## License
This project is licensed via the MIT README.
