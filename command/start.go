package command

import (
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/lodastack/alarm-adapter/adapter"
	"github.com/lodastack/alarm-adapter/config"
	"github.com/lodastack/log"

	"github.com/oiooj/cli"
)

var logBackend *log.FileBackend

var CmdStart = cli.Command{
	Name:        "start",
	Usage:       "启动",
	Description: "启动",
	Action:      runStart,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "c",
			Value: "/etc/alarm-adapter.conf",
			Usage: "配置文件路径，默认位置：/etc/alarm-adapter.conf",
		},
	},
}

func runStart(c *cli.Context) {
	//parse config file
	err := config.ParseConfig(c.String("c"))
	if err != nil {
		log.Fatalf("Parse Config File Error: %s", err.Error())
	}
	//init log setting
	initLog()
	//save pid to file
	ioutil.WriteFile(config.PID, []byte(strconv.Itoa(os.Getpid())), 0744)
	go Notify()

	//start main
	go adapter.Start()
	select {}
}

func initLog() {
	var err error
	logBackend, err = log.NewFileBackend(config.C.Log.Dir)
	if err != nil {
		log.Fatalf("failed to new log backend")
	}
	log.SetLogging(config.C.Log.Level, logBackend)
	logBackend.Rotate(config.C.Log.Logrotatenum, config.C.Log.Logrotatesize)
}

func Notify() {
	message := make(chan os.Signal, 1)

	signal.Notify(message, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGKILL, os.Interrupt)
	<-message
	log.Info("receive signal, exit...")
	logBackend.Flush()
	os.Exit(0)
}
