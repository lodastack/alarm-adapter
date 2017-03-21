package command

import (
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strconv"
	"syscall"

	"github.com/lodastack/alarm-adapter/APIStatus"
	"github.com/lodastack/alarm-adapter/adapter"
	"github.com/lodastack/alarm-adapter/config"
	"github.com/lodastack/alarm-adapter/ping"
	"github.com/lodastack/alarm-adapter/snmp"
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
		cli.StringFlag{
			Name:  "cpuprofile",
			Value: "",
			Usage: "CPU profile",
		},
		cli.StringFlag{
			Name:  "memprofile",
			Value: "",
			Usage: "memory profile",
		},
	},
}

func runStart(c *cli.Context) {
	//parse config file
	err := config.ParseConfig(c.String("c"))
	// Start requested profiling.
	startProfile(c.String("cpuprofile"), c.String("memprofile"))
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
	go ping.Start()
	go api.Start()
	go snmp.Start()
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
	stopProfile()
	os.Exit(0)
}

// prof stores the file locations of active profiles.
var prof struct {
	cpu *os.File
	mem *os.File
}

// startProfile initializes the CPU and memory profile, if specified.
func startProfile(cpuprofile, memprofile string) {
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Errorf("failed to create CPU profile file at %s: %s", cpuprofile, err.Error())
		}
		log.Printf("writing CPU profile to: %s\n", cpuprofile)
		prof.cpu = f
		pprof.StartCPUProfile(prof.cpu)
	}

	if memprofile != "" {
		f, err := os.Create(memprofile)
		if err != nil {
			log.Errorf("failed to create memory profile file at %s: %s", cpuprofile, err.Error())
		}
		log.Printf("writing memory profile to: %s\n", memprofile)
		prof.mem = f
		runtime.MemProfileRate = 4096
	}
}

// stopProfile closes the CPU and memory profiles if they are running.
func stopProfile() {
	if prof.cpu != nil {
		pprof.StopCPUProfile()
		prof.cpu.Close()
		log.Printf("CPU profiling stopped")
	}
	if prof.mem != nil {
		pprof.Lookup("heap").WriteTo(prof.mem, 0)
		prof.mem.Close()
		log.Printf("memory profiling stopped")
	}
}
