package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/creasty/defaults"
	"gopkg.in/yaml.v2"
)

// log.Fatalf использую только до запуска сервера

// Config настройки
type Config struct {
	BindTo string `yaml:"bindTo" default:"http://:8080/sites?search="`
	// MinWaitTime минимальный порог времени ответа, до которого грузим сайт
	MinWaitTime time.Duration `yaml:"minWaitTime" default:"3s"`
	// MaxWaitTime предельное время на ответ
	MaxWaitTime time.Duration `yaml:"maxWaitTime" default:"5s"`
	// EndStopTime через сколько времени прекратить ждать ответов и отдать результат
	EndStopTime time.Duration `yaml:"endStopTime" default:"30s"`
	// со скольки запросов начинать тренировку.
	MidConcurrency int `yaml:"midConcurrency" default:"10"`
	// ConcurrencyMultiplier во сколько раз увеличивать или уменьшать количество одновременных соединений
	ConcurrencyMultiplier float32 `yaml:"concurrencyMultiplier" default:"2.0"`
	// сколько живет запись в кэше
	CacheTTL time.Duration `yaml:"cacheTTL" default:"60s"`
}

var confName string
var verb bool
var help bool
var gen bool
var trace bool

func init() {
	flag.StringVar(&confName, "conf", "", "config file")
	flag.BoolVar(&verb, "v", false, "verbose")
	flag.BoolVar(&help, "h", false, "help")
	flag.BoolVar(&gen, "gen", false, "gen config")
	flag.BoolVar(&trace, "trace", false, "trace flag")
}

func main() {
	flag.Parse()

	conf := Config{}

	// defaults
	if err := defaults.Set(&conf); err != nil {
		log.Print(err)
	}
	// gen def config
	if gen {
		d, _ := yaml.Marshal(&conf)
		fmt.Printf("# default config %v\n%v\n", time.Now(), string(d))
		os.Exit(0)
	}
	// config, если задан из коммандной строки
	if confName != "" {
		yamlFile, err := ioutil.ReadFile(confName)
		if err != nil {
			log.Fatalf("%v:%v", confName, err)
		}
		err = yaml.Unmarshal(yamlFile, &conf)
		if err != nil {
			log.Fatalf("Unmarshal: %v", err)
		}
	}
	if err := conf.runAPI(); err != nil {
		log.Panic(err)
	}
}
