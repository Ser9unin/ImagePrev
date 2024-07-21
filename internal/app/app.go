package app

import (
	"bufio"
	"bytes"
	"fmt"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
)

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

	// в cache Key пишем строку с параметрами и адресом исходного запроса
	// в формате fill/width/height/jpegSource.com/sourceFileName.jpg
	// в cache Value пишем имя файла, с которым он буде храниться на диске
	// в формате width_height_sourceFileName.jpg
	app.cache.Set(paramsStr, filename)
	app.logger.Info(fmt.Sprintf("set cache file: %s", filename))

	srcImage, err := jpeg.Decode(bytes.NewReader(byteImg))
	if err != nil {
		return nil, err
	}

	dstImage := imaging.Fill(srcImage, width, height, imaging.Center, imaging.Lanczos)

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

	// клиенту возвращаем jpeg в виде байт
	return bytesResponse.Bytes(), nil
}

// parseParams достаёт из запроса данные о ширине и высоте, до которых нужно изменить размер,
// а так же имя файла, с которым тот будет сохранен на диске
// в формате width_height_sourceFileName.jpg
func parseParams(paramsStr string) (width, height int, fileName string, err error) {
	splitParams := strings.Split(paramsStr, "/")
	width, err = strconv.Atoi(splitParams[1])
	if err != nil {
		return 0, 0, "", fmt.Errorf("wrong width data: %s", err)
	}
	height, err = strconv.Atoi(splitParams[2])
	if err != nil {
		return 0, 0, "", fmt.Errorf("wrong height data: %s", err)
	}

	sLen := len(splitParams) - 1
	fileName = splitParams[1] + "_" + splitParams[2] + "_" + splitParams[sLen]

	return width, height, fileName, nil
}

func fileStorage(bytesResponse bytes.Buffer, filename string) error {
	// проверяем есть лм файл с таким названием на диске
	file, err := os.Open("../storage/" + filename)
	if err != nil {
		// если такого файла нет, создаём новый
		err = saveFileOnDisk(bytesResponse.Bytes(), filename)
		if err != nil {
			return err
		}
		return nil
	} else {
		// если есть файл с таким названием, то сравниваем байты файла на диске с байтами,
		// которые у нас получились после изменения размера изображения
		fileBytes, err := io.ReadAll(file)
		// если файл не удалось прочитать, пробуем его перезаписать
		if err != nil {
			err = saveFileOnDisk(bytesResponse.Bytes(), filename)
			if err != nil {
				return err
			}
		}
		ok := bytes.Equal(fileBytes, bytesResponse.Bytes())
		if !ok {
			// если байты не совпадают, перезаписываем файл
			err = saveFileOnDisk(bytesResponse.Bytes(), filename)
			if err != nil {
				return err
			}
		} else {
			// если байты совпадают, закрываем файл
			file.Close()
		}
		return nil
	}
}

func saveFileOnDisk(fileBytes []byte, filename string) error {
	file, err := os.Create("../storage/" + filename)
	if err != nil {
		return fmt.Errorf("can't create file: %s", err)
	}
	// записываем jpeg с новыми размерами
	_, err = file.Write(fileBytes)
	if err != nil {
		return fmt.Errorf("can't create file: %s", err)
	}
	// закрываем файл после использования
	defer file.Close()
	return nil
}

// ProxyRequest проксирует header исходного запроса к источнику откуда будет скачиваться изображение,
// запускает скачивание файла от внешнего сервиса
// (вероятно скачивание правильнее выделить в отдельную функцию, а от ProxyRequest забрать только header)
func (app *App) ProxyRequest(targetUrl string, initHeaders http.Header) ([]byte, int, error) {
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
		app.logger.Info("JPEG image receiving")

		// скачиваем ответ через буфер, что бы не получить слишком большой файл
		//  и прекратить чтение при превышении лимита 100 мегабайт
		data, status, err := app.responseBufferReader(targetResp.Body)
		if err != nil {
			return nil, status, err
		} else {
			app.logger.Info("JPEG image received")
			return data, status, nil
		}
	} else {
		return nil, http.StatusUnsupportedMediaType, fmt.Errorf("not a JPEG image")
	}
}

// responseBufferReader читает файл из источника по 1 килобайту,
// до конца файла или достижения лимита в 100 мегабайт.
// Если лимит превышен возвращает то, что было вычитано и ошибку.
func (app *App) responseBufferReader(targetBody io.ReadCloser) ([]byte, int, error) {
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
		app.logger.Info(fmt.Sprintf("Received %d bytes", bytesRead))
	}
	return buffer, http.StatusOK, nil
}
