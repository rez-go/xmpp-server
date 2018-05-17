package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func main() {
	srv, err := New(&Config{
		Name:   "test",
		Domain: "localhost",
		Port:   "5222",
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Server start")
	go srv.Serve()

	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	var forceStop bool
	stopTimeout := time.NewTimer(10 * time.Second)
	stopTimeout.Stop()

mainloop:
	for {
		select {
		case sig := <-signalCh:
			if forceStop {
				log.Infof("Got signal %v. Forcing exit.", sig)
				break mainloop
			}
			log.Info("Got signal ", sig)
			srv.Stop()
			forceStop = true
			stopTimeout.Reset(10 * time.Second)
		case <-stopTimeout.C:
			log.Info("Shutdown timeout. Forcing exit.")
			break mainloop
		case <-srv.DoneCh:
			break mainloop
		}
	}

	log.Info("Exit.")
}
