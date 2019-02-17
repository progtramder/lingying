package main

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"ziphttp"
)

type Config struct {
	Cert string `yaml:"cert_path"`
	Key  string `yaml:"key_path"`
	Avatar string `yaml:"avatar_path"`
}

func main() {
	cpus := runtime.NumCPU()
	p := flag.Int("p", cpus - 2, "number of cpu to run on")
	ds := flag.String("ds", "localhost", "ip address of db server")
	flag.Parse()
	runtime.GOMAXPROCS(*p)

	path, _:= filepath.Abs(filepath.Dir(os.Args[0]))
	config := Config{}
	setting, err := ioutil.ReadFile(path + "/config.yaml")
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	yaml.Unmarshal(setting, &config)

	fmt.Println("Loading database...")
	err = initDb(*ds)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println("Done.")

	time.AfterFunc(time.Second, IntervalHandler)

	fmt.Println("Starting server...")
	http.Handle("/avatar/",
		http.StripPrefix("/avatar/", http.FileServer(http.Dir(config.Avatar))))
	http.HandleFunc("/cancel", handleCancel)
	http.HandleFunc("/course", handleCourse)
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/register", handleRegister)
	http.HandleFunc("/authorize", handleAuthorize)
	http.HandleFunc("/get-timer", handleGetTimer)
	http.HandleFunc("/set-timer", handleSetTimer)
	http.HandleFunc("/register-info", handleRegisterInfo)
	http.HandleFunc("/register-history", handleRegisterHistory)

	srv := &http.Server{
		Addr:        ":443",
		ReadTimeout  : 5 * time.Second,
		WriteTimeout : 5 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServeTLS(config.Cert, config.Key); err != nil {
			log.Fatal(err)
		}
	}()

	time.Sleep(time.Second)
	fmt.Println("Done.")

	ziphttp.CmdLineLoop(prompt, func(input string) int {
		handler, ok := CmdLineHandler[input]
		if ok {
			return handler.Handle()
		}

		return Continue()
	})
}
