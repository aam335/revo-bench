package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var u *url.URL

type Reply struct {
	Result   bool           `json:"result"`
	Error    error          `json:"error"`
	Query    string         `json:"query"`
	Sites    map[string]int `json:"sites"`
	ExecTime time.Duration  `json:"nanoseconds"`
}

func (r *Reply) WriteReply(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json") // ответ только json
	// можно подрулить http response code
	// if r.Error == nil {
	// 	w.WriteHeader(http.StatusOK)
	// }
	_ = json.NewEncoder(w).Encode(r) // проверка кода возвращения возвращающей код процедуры
}

func (conf *Config) api(w http.ResponseWriter, r *http.Request) {
	reply := Reply{}
	rawQuery := r.FormValue(u.RawQuery)
	stt := time.Now()
	oneLoop := true // break=выход по ошибке
	for oneLoop {
		oneLoop = false
		if rawQuery == "" {
			reply.Error = fmt.Errorf("no '%v' variable in query", u.RawQuery)
			break // прекращение обработки тела for {...}
		}
		reply.Query = rawQuery

		// Request the HTML page.
		yaUrl := fmt.Sprintf(baseYandexURL, url.QueryEscape(rawQuery))
		res, err := http.Get(yaUrl)
		if err != nil {
			reply.Error = err
			break
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			reply.Error = fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
			break
		}
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			reply.Error = err
			break
		}
		response := parseYandexResponse(body)
		// log.Printf("%#v", response)
		if response.Error != nil {
			reply.Error = response.Error
			break
		}
		reply.Sites = make(map[string]int)
		meWait := 0 // счетчик взведенных тренеров, использование аналогично waitgroup
		collectChan := make(chan Trainer)
		stopChan := make(chan interface{})
		for _, item := range response.Items {
			if _, ok := reply.Sites[item.Host]; ok {
				continue
			}
			// проверяю кэш
			if reply.Sites[item.Host] = cache.Get(item.Host, conf.CacheTTL); reply.Sites[item.Host] > 0 {
				continue
			}
			meWait += 1
			Train(item, conf, collectChan, stopChan)
			// break
		}
		timer := time.NewTimer(conf.EndStopTime - conf.MaxWaitTime) // оповещение всех тренеров, что цикл чтения - последний
		for cnt := 0; cnt < meWait; cnt++ {
			select {
			case <-timer.C:
				close(stopChan) // оповещаю всех тренеров, что пора заканчивать
				// всем слушателям приходит close event
			case trainResult := <-collectChan:
				reply.Sites[trainResult.Site] = trainResult.RequestsLimit
				if trainResult.RequestsLimit > 0 { // в кэш только позитив
					cache.Put(trainResult.Site, trainResult.RequestsLimit) // сохраняю в кэш
				}
				log.Printf("%#v", trainResult)
			}
		}
		reply.Result = true // дошли до сюда - все выполнилось как надо
	}
	reply.ExecTime = time.Since(stt)
	reply.WriteReply(w)
}

func (conf *Config) runAPI() error {
	var err error
	u, err = url.Parse(conf.BindTo)
	if err != nil {
		panic(err)
	}
	if u.Path == "" {
		u.Path = "/"
	}
	if u.Host == "" {
		u.Host = ":8080"
	}
	if u.RawQuery == "" {
		u.RawQuery = "search"
	}
	u.RawQuery = strings.TrimSuffix(u.RawQuery, "=")
	http.HandleFunc(u.Path, conf.api)
	return http.ListenAndServe(u.Host, nil)
}
