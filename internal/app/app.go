package app

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
)

var storagePath = "./internal/storage/"

type App struct {
	cache  Cache
	logger Logger
}

type Cache interface {
	Set(key string, value interface{}) bool
	Get(key string) (interface{}, bool)
	Clear()
}
type Logger interface {
	Info(msg string)
	Error(msg string)
	Debug(msg string)
	Warn(msg string)
}

func New(cache Cache, logger Logger) *App {
	return &App{cache: cache, logger: logger}
}

func (app *App) Set(key string, value interface{}) bool {
	return app.cache.Set(key, value)
}

func (app *App) Get(key string) (interface{}, bool) {
	return app.cache.Get(key)
}

func (app *App) Clear() {
	app.cache.Clear()
}

func (app *App) Fill(byteImg []byte, paramsStr string) ([]byte, error) {
	width, height, filename, err := parseParams(paramsStr)
	if err != nil {
		return nil, err
	}

	rawJpeg := bytes.NewReader(byteImg)

	srcImage, err := jpeg.Decode(rawJpeg)
	if err != nil {
		// при этой ошибке не всегда есть реальная проблема, повторная попытка декодирования помогает
		if err.Error() == "invalid JPEG format: too many coefficients" {
			srcImage, err = jpeg.Decode(rawJpeg)
			app.logger.Info(err.Error())
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	dstImage := imaging.Fit(srcImage, width, height, imaging.Lanczos)

	var bytesResponse bytes.Buffer
	err = jpeg.Encode(&bytesResponse, dstImage, nil)
	if err != nil {
		return nil, err
	}

	app.logger.Info(fmt.Sprintf("saving file on disk: %s", filename))
	// кэшуруем файлы на диске
	err = fileStorage(bytesResponse, filename)
	// если файл сохранить не удалесь, возвращаем клиенту картинку,
	// а ошибку сохранения возвращаем на сервер и там логируем
	if err != nil {
		app.logger.Error(fmt.Sprintf("failed to save file: %s", filename))
		return bytesResponse.Bytes(), err
	}
	app.logger.Info(fmt.Sprintf("file saved disk: %s", filename))

	// в cache Key пишем строку с параметрами и адресом исходного запроса
	// в формате fill/width/height/jpegSource.com/sourceFileName.jpg
	// в cache Value пишем имя файла, с которым он буде храниться на диске
	// в формате width_height_sourceFileName.jpg.
	app.cache.Set(paramsStr, filename)
	app.logger.Info(fmt.Sprintf("set cache file: %s", filename))

	// клиенту возвращаем jpeg в виде байт
	return bytesResponse.Bytes(), nil
}

// parseParams достаёт из запроса данные о ширине и высоте, до которых нужно изменить размер,
// а так же имя файла, с которым тот будет сохранен на диске
// в формате width_height_sourceFileName.jpg.
func parseParams(paramsStr string) (width, height int, fileName string, err error) {
	splitParams := strings.Split(paramsStr, "/")
	if len(splitParams) < 4 {
		return 0, 0, "", fmt.Errorf("not enough params")
	}
	width, err = strconv.Atoi(splitParams[2])
	if err != nil {
		return 0, 0, "", fmt.Errorf("wrong width data: %w", err)
	}
	height, err = strconv.Atoi(splitParams[3])
	if err != nil {
		return 0, 0, "", fmt.Errorf("wrong height data: %w", err)
	}
	if width < 1 || height < 1 {
		return 0, 0, "", fmt.Errorf("width or height less than 1")
	}
	sLen := len(splitParams) - 1
	fileName = splitParams[2] + "x" + splitParams[3] + "_" + splitParams[sLen]
	return width, height, fileName, nil
}

func fileStorage(bytesResponse bytes.Buffer, filename string) error {
	_, err := os.Stat(storagePath)
	if os.IsNotExist(err) {
		log.Println("Папки не существует, создаём...")
		err := os.Mkdir(storagePath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("ошибка создания папки: %w", err)
		}
		log.Println("Папка создана успешно")
	}

	filePath := storagePath + filename

	err = saveFileOnDisk(bytesResponse.Bytes(), filePath)
	if err != nil {
		return err
	}
	return nil
}

func saveFileOnDisk(fileBytes []byte, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("can't create file: %w", err)
	}
	// записываем jpeg с новыми размерами
	_, err = file.Write(fileBytes)
	if err != nil {
		return fmt.Errorf("can't create file: %w", err)
	}
	// закрываем файл после использования
	defer file.Close()
	return nil
}

// ProxyRequest проксирует header исходного запроса к источнику откуда будет скачиваться изображение.
func (app *App) ProxyHeader(targetURL string, initHeader http.Header) (*http.Request, int, error) {
	// Создаем новый запрос к целевому сервису
	targetURLhttps := "https://" + targetURL
	targetReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, targetURLhttps, nil)
	app.logger.Info(targetURLhttps)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error creating request: %w", err)
	}

	// Копируем все заголовки из исходного запроса в новый
	for name, values := range initHeader {
		for _, value := range values {
			targetReq.Header.Add(name, value)
		}
	}
	return targetReq, http.StatusOK, nil
}

func (app *App) FetchExternalData(targetReq *http.Request) ([]byte, int, error) {
	// Отправляем запрос и обрабатываем ответ
	transport := &http.Transport{
		DisableKeepAlives: false,
	}

	client := &http.Client{Transport: transport}

	targetResp, err := client.Do(targetReq)
	if err != nil {
		app.logger.Error(err.Error())
		app.logger.Info(targetReq.RequestURI)
		targetReq.URL.Scheme = "http"
		targetResp, err = client.Do(targetReq)
		if err != nil {
			app.logger.Error(fmt.Sprintf("Status %d, %s", http.StatusBadGateway, err.Error()))
			return nil, http.StatusBadGateway, fmt.Errorf("error sending request")
		}
		defer func() {
			if err := targetResp.Body.Close(); err != nil {
				return
			}
		}()
	}
	defer func() {
		if err := targetResp.Body.Close(); err != nil {
			return
		}
	}()

	// Проверяем, что внешний сервис не ответил 404
	if targetResp.StatusCode == http.StatusNotFound {
		return nil, targetResp.StatusCode, fmt.Errorf("content not found")
	}

	// Проверяем, что внешний сервис отправляет jpeg, если да, то читаем его через буфер.
	contentType := targetResp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/jpeg") {
		return nil, http.StatusUnsupportedMediaType, fmt.Errorf("not a JPEG image")
	}
	app.logger.Info("JPEG image receiving")

	// скачиваем ответ через буфер, что бы не получить слишком большой файл
	//  и прекратить чтение при превышении лимита 100 мегабайт
	result, status, err := app.responseBufferReader(targetResp.Body)
	if err != nil {
		return nil, status, err
	}
	app.logger.Info("JPEG image received")
	return result, http.StatusOK, nil
}

// responseBufferReader читает файл из источника по 1 килобайту,
// до конца файла или достижения лимита в 100 мегабайт.
// Если лимит превышен возвращает то, что было вычитано и ошибку.
func (app *App) responseBufferReader(targetBody io.ReadCloser) ([]byte, int, error) {
	reader := bufio.NewReader(targetBody)

	result := make([]byte, 0, 104857600)
	buffer := bytes.NewBuffer(result)

	// лимит 100 мегабайт, маловероятно что jpeg будет весить больше,
	// если будет превышение возможно там не jpeg замаскированный под jpeg.
	var limitBytes int64 = 104857600

	bytesRead, err := io.CopyN(buffer, reader, 104857600)

	if errors.Is(err, io.EOF) {
		result = buffer.Bytes()
		app.logger.Info(fmt.Sprintf("Received %d bytes", len(result)))
	} else if err != nil {
		if bytesRead > limitBytes {
			app.logger.Info(fmt.Sprintf("Received %d bytes", len(result)))
			result = buffer.Bytes()
			return result, http.StatusRequestEntityTooLarge, fmt.Errorf("data exceed limit")
		}
		app.logger.Info(fmt.Sprintf("Received %d bytes", len(result)))
		return nil, http.StatusNotFound, fmt.Errorf("error reading request body: %w", err)
	}

	return result, http.StatusOK, nil
}
