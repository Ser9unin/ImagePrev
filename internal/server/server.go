package server

import (
	"context"
	"net/http"

	"github.com/Ser9unin/ImagePrev/internal/config"
)

type Server struct {
	srv    *http.Server
	router *http.ServeMux
	app    App
	logger Logger
}

type Logger interface {
	Info(msg string)
	Error(msg string)
	Debug(msg string)
	Warn(msg string)
}

type App interface {
	Set(key string, value interface{}) bool
	Get(key string) (interface{}, bool)
	Clear()
	Fill(byteImg []byte, paramsStr string) ([]byte, error)
	ProxyRequest(url string, headers http.Header) ([]byte, int, error)
}

func (s *Server) NewServer(cfg config.Config, app App, logger Logger) *Server {
	router := s.NewRouter(app, logger)

	srv := &http.Server{
		Addr:    cfg.Server.Port,
		Handler: router,
	}

	return &Server{srv, router, app, logger}
}

func (s *Server) NewRouter(app App, logger Logger) *http.ServeMux {
	mux := http.NewServeMux()

	mw := func(next http.HandlerFunc) http.HandlerFunc {
		return HttpLogger(CheckHttpMethod(next))
	}

	a := newAPI(app, logger)

	mux.HandleFunc("/", mw(a.greetings))
	mux.HandleFunc("/fill/", mw(a.fill))

	return mux
}

func (s *Server) Run(ctx context.Context) error {
	return s.srv.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
