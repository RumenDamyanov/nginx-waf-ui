package handler

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"github.com/RumenDamyanov/nginx-waf-ui/internal/api"
	"github.com/RumenDamyanov/nginx-waf-ui/internal/session"
)

//go:embed templates/*
var templateFS embed.FS

// Handler holds web handler dependencies.
type Handler struct {
	client  *api.Client
	store   *session.Store
	logger  *slog.Logger
	tmpl    *template.Template
	adminPw string // initial admin password (bcrypt would be better, but keep simple for v0.1)
}

// New creates a Handler. adminPassword is the password for the "admin" user.
func New(client *api.Client, store *session.Store, logger *slog.Logger, adminPassword string) *Handler {
	funcMap := template.FuncMap{
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			return s[:n] + "..."
		},
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/layouts/*.html", "templates/pages/*.html"))
	return &Handler{
		client:  client,
		store:   store,
		logger:  logger,
		tmpl:    tmpl,
		adminPw: adminPassword,
	}
}

// RegisterRoutes attaches handlers to the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /login", h.loginPage)
	mux.HandleFunc("POST /login", h.loginSubmit)
	mux.HandleFunc("POST /logout", h.logout)
	mux.HandleFunc("GET /", h.requireAuth(h.dashboard))
	mux.HandleFunc("GET /lists", h.requireAuth(h.listsPage))
	mux.HandleFunc("GET /lists/{name}", h.requireAuth(h.listDetail))
	mux.HandleFunc("POST /lists/{name}/entries", h.requireAuth(h.addEntry))
	mux.HandleFunc("POST /lists/{name}/entries/{ip}/delete", h.requireAuth(h.removeEntry))
	mux.HandleFunc("POST /reload", h.requireAuth(h.triggerReload))
	mux.HandleFunc("GET /health", h.health)
}

type pageData struct {
	Title    string
	Username string
	CSRF     string
	Flash    string
	Data     any
}

func (h *Handler) render(w http.ResponseWriter, name string, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		h.logger.Error("template render", "error", err, "template", name)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess := h.store.Get(r)
		if sess == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func (h *Handler) loginPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, "login.html", pageData{Title: "Login", CSRF: generateCSRF()})
}

func (h *Handler) loginSubmit(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if username == "admin" && password == h.adminPw {
		h.store.Create(w, username)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	h.render(w, "login.html", pageData{
		Title: "Login",
		Flash: "Invalid username or password",
		CSRF:  generateCSRF(),
	})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	h.store.Destroy(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	sess := h.store.Get(r)
	lists, err := h.client.GetLists()
	if err != nil {
		h.logger.Error("fetch lists", "error", err)
		lists = nil
	}

	totalEntries := 0
	for _, l := range lists {
		totalEntries += l.Entries
	}

	data := map[string]any{
		"Lists":        lists,
		"TotalLists":   len(lists),
		"TotalEntries": totalEntries,
		"APIError":     err,
	}

	h.render(w, "dashboard.html", pageData{
		Title:    "Dashboard",
		Username: sess.Username,
		Data:     data,
	})
}

func (h *Handler) listsPage(w http.ResponseWriter, r *http.Request) {
	sess := h.store.Get(r)
	lists, err := h.client.GetLists()
	if err != nil {
		h.logger.Error("fetch lists", "error", err)
	}
	h.render(w, "lists.html", pageData{
		Title:    "IP Lists",
		Username: sess.Username,
		Data:     map[string]any{"Lists": lists, "Error": err},
	})
}

func (h *Handler) listDetail(w http.ResponseWriter, r *http.Request) {
	sess := h.store.Get(r)
	name := r.PathValue("name")
	detail, err := h.client.GetList(name)
	if err != nil {
		h.logger.Error("fetch list", "error", err, "name", name)
	}
	h.render(w, "list_detail.html", pageData{
		Title:    "List: " + name,
		Username: sess.Username,
		CSRF:     generateCSRF(),
		Data:     map[string]any{"Detail": detail, "Name": name, "Error": err},
	})
}

func (h *Handler) addEntry(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ip := strings.TrimSpace(r.FormValue("ip"))
	if ip == "" {
		http.Redirect(w, r, "/lists/"+name, http.StatusSeeOther)
		return
	}
	if err := h.client.AddEntry(name, ip); err != nil {
		h.logger.Error("add entry", "error", err, "list", name, "ip", ip)
	}
	http.Redirect(w, r, "/lists/"+name, http.StatusSeeOther)
}

func (h *Handler) removeEntry(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ip := r.PathValue("ip")
	if err := h.client.RemoveEntry(name, ip); err != nil {
		h.logger.Error("remove entry", "error", err, "list", name, "ip", ip)
	}
	http.Redirect(w, r, "/lists/"+name, http.StatusSeeOther)
}

func (h *Handler) triggerReload(w http.ResponseWriter, r *http.Request) {
	if err := h.client.Reload(); err != nil {
		h.logger.Error("reload", "error", err)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func generateCSRF() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
