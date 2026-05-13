package app

import (
    "database/sql"
    "fmt"
    "net/http"
    "os"
    "path/filepath"
    "time"

    "github.com/alexedwards/scs/v2"
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"

    "slatessh/backend/internal/auth"
    "slatessh/backend/internal/config"
    "slatessh/backend/internal/connections"
    crypto2 "slatessh/backend/internal/crypto"
    "slatessh/backend/internal/db"
    "slatessh/backend/internal/settings"
    "slatessh/backend/internal/transfers"
    "slatessh/backend/internal/users"
    "slatessh/backend/internal/ws"
)

type App struct {
    cfg             config.Config
    db              *sql.DB
    router          http.Handler
    authService     *auth.Service
    authHandler     *auth.Handler
    settingsHandler *settings.Handler
    connections     *connections.Handler
    transfers       *transfers.Handler
    wsHub           *ws.Hub
}

func New() (*App, error) {
    cfg, err := config.Load()
    if err != nil {
        return nil, err
    }

    database, err := db.Open(cfg.DBPath)
    if err != nil {
        return nil, err
    }
    if err := db.Migrate(database); err != nil {
        return nil, err
    }

    sessions := scs.New()
    sessions.Lifetime = 365 * 24 * time.Hour
    sessions.Cookie.Name = "slatessh_session"
    sessions.Cookie.HttpOnly = true
    sessions.Cookie.Path = "/"
    sessions.Cookie.SameSite = http.SameSiteLaxMode

    usersRepo := users.NewRepository(database)
    cryptoService, err := crypto2.New(cfg.EncryptionKey)
    if err != nil {
        return nil, err
    }
    connectionsRepo := connections.NewRepository(database)
    connectionsService := connections.NewService(connectionsRepo, cryptoService)
    authService := auth.NewService(usersRepo, sessions)

    application := &App{
        cfg:             cfg,
        db:              database,
        authService:     authService,
        authHandler:     auth.NewHandler(authService),
        settingsHandler: settings.NewHandler(settings.NewRepository(database)),
        connections:     connections.NewHandler(connectionsService),
        transfers:       transfers.NewHandler(database),
        wsHub:           ws.NewHub(connectionsService, authService),
    }
    application.router = sessions.LoadAndSave(application.routes())
    return application, nil
}

func (a *App) Run() error {
    address := fmt.Sprintf("%s:%d", a.cfg.Host, a.cfg.Port)
    return http.ListenAndServe(address, a.router)
}

func (a *App) routes() http.Handler {
    router := chi.NewRouter()
    router.Use(middleware.RequestID)
    router.Use(middleware.RealIP)
    router.Use(middleware.Logger)
    router.Use(middleware.Recoverer)

    router.Get("/api/v1/auth/needs-setup", a.authHandler.NeedsSetup)
    router.Post("/api/v1/auth/setup", a.authHandler.SetupAdmin)
    router.Post("/api/v1/auth/login", a.authHandler.Login)
    router.Post("/api/v1/auth/logout", a.authHandler.Logout)
    router.Get("/api/v1/auth/status", a.authHandler.Status)
    router.Post("/api/v1/auth/change-password", a.authHandler.ChangePassword)

    router.Group(func(protected chi.Router) {
        protected.Use(func(next http.Handler) http.Handler { return auth.RequireAuth(a.authService, next) })
        protected.Get("/api/v1/connections", a.connections.List)
        protected.Post("/api/v1/connections", a.connections.Create)
        protected.Post("/api/v1/connections/test-unsaved", a.connections.TestUnsaved)
        protected.Get("/api/v1/connections/{id}", a.connections.Get)
        protected.Put("/api/v1/connections/{id}", a.connections.Update)
        protected.Delete("/api/v1/connections/{id}", a.connections.Delete)
        protected.Post("/api/v1/connections/{id}/test", a.connections.TestSaved)

        protected.Get("/api/v1/settings", a.settingsHandler.GetAll)
        protected.Post("/api/v1/settings", a.settingsHandler.UpdateMany)

        protected.Post("/api/v1/transfers", a.transfers.Initiate)
        protected.Get("/api/v1/transfers", a.transfers.List)
        protected.Get("/api/v1/transfers/{taskId}", a.transfers.Get)
        protected.Post("/api/v1/transfers/{taskId}/cancel", a.transfers.Cancel)

        protected.Get("/api/v1/files/download", a.wsHub.ServeDownload)
        protected.Post("/api/v1/files/upload", a.wsHub.ServeUpload)
        protected.Handle("/ws", a.wsHub)
    })

    frontendDir := filepath.Clean(filepath.Join(".", "frontend"))
    if _, err := os.Stat(filepath.Join(frontendDir, "index.html")); err == nil {
        router.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir(filepath.Join(frontendDir, "assets")))))
        router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
            http.ServeFile(w, r, filepath.Join(frontendDir, "index.html"))
        })
    }

    return router
}
