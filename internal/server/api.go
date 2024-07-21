package server

import (
	"net/http"
	"net/url"
	"os"
	"strings"
)

type api struct {
	app    App
	logger Logger
}

func newAPI(app App, logger Logger) api {
	return api{
		app:    app,
		logger: logger,
	}
}

func (a *api) greetings(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("This is my previewer!"))
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
		filePath := "../storage/" + cachePath.(string)
		fileFromDisc, err := os.ReadFile(filePath)
		if err != nil {
			a.logger.Error(err.Error())
		}
		a.logger.Info("image get from cache")
		responseImage(w, r, http.StatusOK, fileFromDisc)
	} else {
		targetURL := parseTargetUrl(paramsStr.Path)
		externalData, httpStatus, err := a.app.ProxyRequest(targetURL, r.Header)
		if err != nil {
			ErrorJSON(w, r, httpStatus, err, "fail proxy request")
		}
		response, err := a.app.Fill(externalData, paramsStr.Path)
		if err != nil {
			a.logger.Error(err.Error())
		}
		responseImage(w, r, http.StatusOK, response)
	}
}

func parseTargetUrl(paramsStr string) string {
	splitParams := strings.Split(paramsStr, "/")
	return strings.Join(splitParams[:2], "/")
}
