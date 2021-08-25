package main

import (
	"net/http"
	"time"
)

type Trainer struct {
	Site          string
	RequestsLimit int
}

// для масштабирования переделывается на mq по вкусу с изменением логики таймаутов на автономную
// при желании можно разделить и запросы в отдельные микросервисы, но это будет нужно при космических rps

func Train(item responseItem, conf *Config, collectChan chan Trainer, stopChan chan interface{}) {
	go func() {
		parallels := conf.MidConcurrency // со скольки запросов начинаем
		lastKnownGood := -1
		contLoop := true
		resChan := make(chan time.Duration)
		mul := true
		firstPass := true
		for contLoop {
			// log.Printf(">%v: %v", item.Url, parallels)
			for cnt := 0; cnt < parallels; cnt++ {
				// можно упростить до true/false, но в этом случае пропадает
				// репрезентативность ответов и мы не получим раннего положительного
				// "когда все запросы выполнялись более [нижний предел]"
				// вариант обрыва соединения ранее таймаута реализовывать не стал
				// буду стопать заранее.
				go func() {
					stt := time.Now()
					client := http.Client{Timeout: conf.MaxWaitTime}
					resp, err := client.Get(item.Url)
					// log.Printf("Got:%v %v", err, resp.StatusCode)
					if err != nil || resp.StatusCode != http.StatusOK { // ловим только 200, остальное аналогично, по желанию
						resChan <- time.Duration(0)
						return
					}
					resChan <- time.Since(stt)
					resp.Body.Close() // можно что-то делать с телом тут
				}()
			}

			resOk := true
			resAllAboveMinLim := true
			for cnt := 0; cnt < parallels; cnt++ {
				select {
				case <-stopChan: // остановка
					contLoop = false
				case res := <-resChan:
					if res == 0 {
						resOk = false
					}
					if res < conf.MinWaitTime {
						resAllAboveMinLim = false
					}
				}
			}
			// смотрю куда скалить от начального
			// если сразу все плохо = вниз и ответ - первый рабочий
			// если первый хорошо, то или по "все больше минимального, но не таймаут"
			// или до первого с ошибками или до stop от мастера
			// Коэф скалирования можно рассчитывать из среднего времени ответа, не реализовано
			if firstPass && !resOk {
				mul = false
			}
			firstPass = false

			if mul {
				if resOk { // все хорошо, трейн засчитан
					lastKnownGood = parallels
					if resAllAboveMinLim { // все отвечало дольшн мин. ответа, тут можно добавить интеллекта
						contLoop = false
					}
				}
				if !resOk && lastKnownGood > 0 { // мы трейним вверх, ответ не проходит, принимаем предыдущий за правду
					contLoop = false
				}
				parallels = int(float32(parallels) * conf.ConcurrencyMultiplier)
				continue
			}
			if resOk { // трейним вниз, первый положительный - результат
				lastKnownGood = parallels
				contLoop = false
			}
			parallels = int(float32(parallels) / conf.ConcurrencyMultiplier)
			if parallels < 1 { // доделили до 0, опрашивать не получается
				contLoop = false
			}
		}
		collectChan <- Trainer{Site: item.Host, RequestsLimit: lastKnownGood}
		// log.Printf("<%v: %v", item.Url, lastKnownGood)
	}()
}
