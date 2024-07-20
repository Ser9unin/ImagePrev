package server

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
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
		return
	}

	width, height, targetURL, fileName, err := parseUrlParams(paramsStr.Path)
	if err != nil {
		log.Fatal(err)
	}

	cacheData, ok := a.app.Get(fileName)
	if ok {
		a.logger.Info("image get from cache")
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(cacheData)
		return
	} else {
		externalData, httpStatus, err := a.app.ProxyRequest(targetURL, r.Header)
		if err != nil {
			ErrorJSON(w, r, httpStatus, err, "fail proxy request")
		}

		response, err := a.app.Fill(externalData, width, height)
		if err != nil {
			a.logger.Error(err.Error())
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(response)
	}
}

func parseUrlParams(paramsStr string) (width, height int, targetURL, fileName string, err error) {
	splitParams := strings.Split(paramsStr, "/")
	width, err = strconv.Atoi(splitParams[1])
	if err != nil {
		return 0, 0, "", "", fmt.Errorf("wrong width data: %s", err)
	}
	height, err = strconv.Atoi(splitParams[2])
	if err != nil {
		return 0, 0, "", "", fmt.Errorf("wrong height data: %s", err)
	}

	targetURL = strings.Join(splitParams[:2], "/")
	sLen := len(splitParams) - 1

	fileName = splitParams[1] + "_" + splitParams[2] + "_" + splitParams[sLen]

	return width, height, targetURL, fileName, nil
}
