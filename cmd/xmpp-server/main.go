package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

func main() {
	srv, err := New(&Config{
		Name:   "test",
		Domain: "localhost",
		Port:   "5222",
	})
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("Server start")
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
				logrus.Infof("Got signal %v. Forcing exit.", sig)
				break mainloop
			}
			logrus.Info("Got signal ", sig)
			srv.Stop()
			forceStop = true
			stopTimeout.Reset(10 * time.Second)
		case <-stopTimeout.C:
			logrus.Info("Shutdown timeout. Forcing exit.")
			break mainloop
		case <-srv.DoneCh:
			break mainloop
		}
	}

	logrus.Info("Exit.")
}
