package server

import (
	"net/http"
	"net/url"
	"os"
	"strings"
)

type api struct {
	app         App
	logger      Logger
	storagePath string
}

func newAPI(app App, logger Logger) api {
	return api{
		app:         app,
		logger:      logger,
		storagePath: "./internal/storage/",
	}
}

func (a *api) greetings(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("<h1>This is my previewer!</h1>"))
}

func (a *api) fill(w http.ResponseWriter, r *http.Request) {
	urlString := r.URL.String()
	paramsStr, err := url.Parse(urlString)
	if err != nil {
		a.logger.Error(err.Error())
		ErrorJSON(w, r, http.StatusBadRequest, err, "not correct path")
	}
	cachePath, ok := a.app.Get(paramsStr.Path)
	if ok {
		filePath := a.storagePath + cachePath.(string)
		fileFromDisc, err := os.ReadFile(filePath)
		if err != nil {
			a.logger.Error(err.Error())
			a.logger.Info("image not found on disk")
			a.externalUpload(w, r, paramsStr.Path)
		} else {
			a.logger.Info("image get from cache")
			w.Header().Set("Get_from_cache", "1")
			responseImage(w, r, http.StatusOK, fileFromDisc)
		}
	} else {
		a.externalUpload(w, r, paramsStr.Path)
	}
}

func (a *api) externalUpload(w http.ResponseWriter, r *http.Request, paramsStr string) {
	paramsURL := parseTargetURL(paramsStr)
	targetReq, httpStatus, err := a.app.ProxyHeader(paramsURL, r.Header)
	if err != nil {
		a.logger.Error(err.Error())
		ErrorJSON(w, r, httpStatus, err, "fail proxy request header")
		return
	}
	externalData, httpStatus, err := a.app.FetchExternalData(targetReq)
	if err != nil {
		a.logger.Error(err.Error())
		ErrorJSON(w, r, httpStatus, err, "fail fetch data request")
		return
	}
	response, err := a.app.Fill(externalData, paramsStr)
	if err != nil {
		a.logger.Error(err.Error())
		ErrorJSON(w, r, httpStatus, err, "fail fetch data")
		return
	}
	w.Header().Set("get_from_remote_server", "1")
	responseImage(w, r, httpStatus, response)
}

func parseTargetURL(paramsStr string) string {
	splitParams := strings.Split(paramsStr, "/")
	return strings.Join(splitParams[4:], "/")
}
