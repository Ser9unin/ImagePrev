package tests

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
}

func (ts *TestSuite) sendRequest(width, height int, targetURL string) (*http.Response, error) {
	url := fmt.Sprintf("http://image-previewer/fill/%d/%d/%s", width, height, targetURL)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	return http.DefaultClient.Do(req)
}

// удаленный сервер вернул изображение.
func (ts *TestSuite) TestServerOK() {
	// в запросе сохраняю изображение без изменений и сравниваю ответ от сервера с исходником
	res, err := ts.sendRequest(1366, 768, "/nginx/testdata/beaver_cute.jpg")
	defer func() {
		if err := res.Body.Close(); err != nil {
			return
		}
	}()

	ts.Require().NoError(err)
	ts.Require().Equal(res.StatusCode, http.StatusOK)

	val := res.Header.Get("Get_from_remote_server")
	log.Println(res.Header)
	ts.Require().Equal(val, "1")
}

// картинка найдена в кэше.
func (ts *TestSuite) TestImageInCache() {
	// делаю первый запрос на сервер с изменением размера изображения
	resFromServer, err := ts.sendRequest(640, 480, "/nginx/testdata/beaver_cute.jpg")
	defer func() {
		if err := resFromServer.Body.Close(); err != nil {
			return
		}
	}()

	ts.Require().NoError(err)
	ts.Require().Equal(resFromServer.StatusCode, http.StatusOK)
	// должен получить ответ в Header изображение с сервера
	val := resFromServer.Header.Get("Get_from_remote_server")
	ts.Require().Equal(val, "1")

	// повторяю запрос
	resCache, err := ts.sendRequest(640, 480, "/nginx/testdata/beaver_cute.jpg")
	defer func() {
		if err := resCache.Body.Close(); err != nil {
			return
		}
	}()

	ts.Require().NoError(err)
	ts.Require().Equal(resCache.StatusCode, http.StatusOK)
	// должен получить ответ в Header изображение изображеине из cache
	val = resCache.Header.Get("Get_from_cache")
	ts.Require().Equal(val, "1")
}

// удаленный сервер не существует.
func (ts *TestSuite) TestBadGateway() {
	res, err := ts.sendRequest(1366, 768, "/badgateway/testdata/my_marmot.jpg")
	defer func() {
		if err := res.Body.Close(); err != nil {
			return
		}
	}()
	ts.Require().NoError(err)
	ts.Require().Equal(res.StatusCode, http.StatusBadGateway)

	body, err := io.ReadAll(res.Body)
	bodyString := string(body)
	bodyString = strings.TrimSuffix(bodyString, "\n")
	ts.Require().NoError(err)
	ts.Require().Equal(bodyString, `{"details":"fail fetch data request","error":"error sending request"}`)
}

// удаленный сервер существует, но изображение не найдено (404 Not Found).
func (ts *TestSuite) TestImageNotFound() {
	res, err := ts.sendRequest(1366, 768, "/nginx/testdata/my_bober.jpg")
	defer func() {
		if err := res.Body.Close(); err != nil {
			return
		}
	}()

	ts.Require().NoError(err)
	ts.Require().Equal(res.StatusCode, http.StatusNotFound)

	body, err := io.ReadAll(res.Body)
	bodyString := string(body)
	bodyString = strings.TrimSuffix(bodyString, "\n")
	ts.Require().NoError(err)
	ts.Require().Equal(bodyString, `{"details":"fail fetch data request","error":"content not found"}`)
}

// удаленный сервер существует, но изображение не изображение, а скажем, exe-файл.
func (ts *TestSuite) TestInvalidMediaType() {
	res, err := ts.sendRequest(1366, 768, "/nginx/testdata/this_is_text.txt")
	defer func() {
		if err := res.Body.Close(); err != nil {
			return
		}
	}()
	ts.Require().NoError(err)
	ts.Require().Equal(res.StatusCode, http.StatusUnsupportedMediaType)

	body, err := io.ReadAll(res.Body)
	bodyString := string(body)
	bodyString = strings.TrimSuffix(bodyString, "\n")
	ts.Require().NoError(err)
	ts.Require().Equal(bodyString, `{"details":"fail fetch data request","error":"not a JPEG image"}`)
}

// изображение меньше, чем нужный размер.
func (ts *TestSuite) TestSize() {
	res, err := ts.sendRequest(0, 0, "/nginx/testdata/my_marmot.jpg")
	defer func() {
		if err := res.Body.Close(); err != nil {
			return
		}
	}()
	ts.Require().NoError(err)

	body, err := io.ReadAll(res.Body)
	bodyString := string(body)
	bodyString = strings.TrimSuffix(bodyString, "\n")
	ts.Require().NoError(err)
	ts.Require().Equal(bodyString, `{"details":"fail fetch data","error":"width or height less than 1"}`)
}

func TestIntegration(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
