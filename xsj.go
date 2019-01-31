package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"ziphttp"
)

func main() {
	cpus := runtime.NumCPU()
	p := flag.Int("p", cpus - 2, "number of cpu to run on")
	flag.Parse()
	runtime.GOMAXPROCS(*p)

	fmt.Println("Loading database...")
	err := initDb()
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println("Done.")

	time.AfterFunc(time.Second, timerHandler)

	fmt.Println("Start serving on port = 80")


	http.HandleFunc("/register", handleRegister)
	http.HandleFunc("/cancel", handleCancel)
	http.HandleFunc("/course", handleCourse)
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/register-info", handleRegisterInfo)
	http.HandleFunc("/register-history", handleRegisterHistory)
	http.HandleFunc("/get-timer", handleGetTimer)
	http.HandleFunc("/set-timer", handleSetTimer)

	srv := &http.Server{
		Addr:        ":443",
		ReadTimeout  : 5 * time.Second,
		WriteTimeout : 5 * time.Second,
	}
	go func() {
		path, _:= filepath.Abs(filepath.Dir(os.Args[0]))
		if err := srv.ListenAndServeTLS(path + "/cert", path + "/key"); err != nil {
			log.Fatal(err)
		}
	}()

	time.Sleep(time.Second)

	ziphttp.CmdLineLoop(prompt, func(input string) int {
		handler, ok := CmdLineHandler[input]
		if ok {
			return handler.Handle()
		}

		return Continue()
	})
}
