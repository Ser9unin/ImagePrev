package server

import (
	"context"
	"log"
	"net/http"
	"os"

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
	ProxyHeader(url string, headers http.Header) (*http.Request, int, error)
	FetchExternalData(targetReq *http.Request) ([]byte, int, error)
}

func NewServer(cfg config.Config, app App, logger Logger) *Server {
	router := NewRouter(app, logger)

	srv := &http.Server{

		Addr:    cfg.Server.Host + cfg.Server.Port,
		Handler: router,
	}

	return &Server{srv, router, app, logger}
}

func NewRouter(app App, logger Logger) *http.ServeMux {
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
	err := os.RemoveAll("./internal/storage/")
	if err != nil {
		log.Println("Ошибка при удалении папки:", err)
	}
	return s.srv.Shutdown(ctx)
}
