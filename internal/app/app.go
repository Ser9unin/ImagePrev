package app

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type App struct {
	logger *zap.Logger
}

func (a *App) Set(key string, value []byte) bool { return false }
func (a *App) Get(key string) ([]byte, bool)     { return nil, false }
func (a *App) Clear()                            {}
func (a *App) Fill(byteImg []byte, width int, height int) ([]byte, error) {
	return nil, nil
}

// ProxyRequest проксирует header исходного запроса к источнику откуда будет скачиваться изображение,
// запускает скачивание файла от внешнего сервиса
// (вероятно правильнее выделить в отдельную функцию, а от ProxyRequest забрать только header)
func (a *App) ProxyRequest(targetUrl string, initHeaders http.Header) ([]byte, int, error) {
	// Создаем новый запрос к целевому сервису
	targetReq, err := http.NewRequest(http.MethodGet, targetUrl, nil)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error creating request")
	}

	// Копируем все заголовки из исходного запроса в новый
	for name, values := range initHeaders {
		for _, value := range values {
			targetReq.Header.Add(name, value)
		}
	}

	// Отправляем запрос и обрабатываем ответ
	targetResp, err := http.DefaultClient.Do(targetReq)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error sending request")
	}
	defer targetResp.Body.Close()

	// КАЖЕТСЯ ЭТОТ КУСОК НЕ НУЖЕН, ПРОКСИРУЕМ ТОЛЬКО ЗАГОЛОВКИ ИСХОДНОГО ЗАПРОСА
	// // Копируем заголовки ответа в исходный запрос
	// for name, values := range targetResp.Header {
	// 	for _, value := range values {
	// 		initHeaders.Add(name, value)
	// 	}
	// }

	// Проверяем, что внешний сервис отправляет jpeg, если да, то читаем его через буфер.
	contentType := targetResp.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "image/jpeg") {
		a.logger.Info("JPEG image receiving")

		// скачиваем ответ через буфер, что бы не получить слишком большой файл
		//  и прекратить чтение при превышении лимита 100 мегабайт
		data, status, err := a.responseBufferReader(targetResp.Body)
		if err != nil {
			return nil, status, err
		} else {
			a.logger.Info("JPEG image received")
			return data, status, nil
		}
	} else {
		return nil, http.StatusUnsupportedMediaType, fmt.Errorf("not a JPEG image")
	}
}

func (a *App) responseBufferReader(targetBody io.ReadCloser) ([]byte, int, error) {
	reader := bufio.NewReader(targetBody)
	buffer := make([]byte, 1024)

	// лимит 100 мегабайт, маловероятно что jpeg будет весить больше,
	// если будет превышение возможно там не jpeg замаскированный под jpeg.
	limitBytes := 104857600
	bytesRead := 0
	var err error
	for {
		bytesRead, err = reader.Read(buffer)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, http.StatusNotFound, fmt.Errorf("error reading request body: %w", err)
		}
		if bytesRead > limitBytes {
			return buffer, http.StatusRequestEntityTooLarge, fmt.Errorf("data exceed limit")
		}
		a.logger.Info("Received", zap.Int("bytes", bytesRead))
	}
	return buffer, http.StatusOK, nil
}
