package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"remote-desktop/internal/app"
	"remote-desktop/internal/engine"
	"syscall"
	"time"
)

var version = "development"

func main() {
	data := flag.String("data", ".data", "private data directory")
	key := flag.String("key", ".data/master.key", "AES key path")
	web := flag.String("web", "web/dist", "static files")
	origin := flag.String("origin", "http://127.0.0.1:18080", "development listener origin")
	dev := flag.Bool("dev", false, "enable authenticated loopback development listener")
	socket := flag.String("socket", "", "trusted gateway Unix socket path")
	guacd := flag.String("guacd", "", "optional loopback guacd address in development mode")
	enginePath := flag.String("engine", "", "absolute path to owned guacd binary")
	backup := flag.String("backup", "", "create an offline backup and exit")
	healthPath := flag.String("healthcheck", "", "check an existing service Unix socket and exit")
	probePath := flag.String("probe-socket", "", "test socket listener: 0 live, 3 stale, 1 unknown")
	showVersion := flag.Bool("version", false, "print application version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}
	if *healthPath != "" {
		if err := healthcheck(*healthPath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *probePath != "" {
		os.Exit(probeSocket(*probePath))
	}
	if *backup == "" && (*dev) == (*socket != "") {
		log.Fatal("choose exactly one: --dev or --socket")
	}
	token := ""
	if *dev {
		token = os.Getenv("RD_DEV_TOKEN")
		if len(token) < 24 {
			log.Fatal("set RD_DEV_TOKEN to at least 24 characters")
		}
	}
	if *backup != "" {
		a, err := app.New(app.Config{DataDir: *data, KeyPath: *key, Origin: *origin, Version: version})
		if err != nil {
			log.Fatal(err)
		}
		defer a.Close()
		if err = a.Backup(*backup); err != nil {
			log.Fatal(err)
		}
		return
	}
	runtimeDir, e := filepath.Abs(filepath.Join(*data, "engine"))
	if e != nil {
		log.Fatal(e)
	}
	if *dev {
		if *guacd == "" {
			*guacd = "127.0.0.1:14822"
		}
		host, _, err := net.SplitHostPort(*guacd)
		if err != nil || !net.ParseIP(host).IsLoopback() {
			log.Fatal("development guacd must use a loopback IP")
		}
	} else {
		if *guacd != "" || *enginePath == "" {
			log.Fatal("gateway mode requires an owned --engine on its private socket")
		}
		*guacd = filepath.Join(runtimeDir, "guacd.sock")
	}
	var l net.Listener
	if *dev {
		l, e = net.Listen("tcp", "127.0.0.1:18080")
	} else {
		if _, err := os.Lstat(*socket); err == nil {
			log.Fatal("socket path already exists; confirm stale process before removal")
		}
		l, e = net.Listen("unix", *socket)
		if e == nil {
			e = os.Chmod(*socket, 0660)

		}
	}
	if e != nil {
		log.Fatal(e)
	}
	defer l.Close()
	if !*dev {
		defer os.Remove(*socket)
	}
	a, e := app.New(app.Config{DataDir: *data, KeyPath: *key, WebDir: *web, Origin: *origin, DevToken: token, Guacd: *guacd, Gateway: !*dev, Version: version})
	if e != nil {
		log.Fatal(e)
	}
	defer a.Close()
	var supervisor *engine.Supervisor
	if *enginePath != "" {
		supervisor, e = engine.Start(*enginePath, *guacd, runtimeDir, a.EngineExited)
		if e != nil {
			log.Fatal(e)
		}
		defer supervisor.Stop()
		readyCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		e = supervisor.WaitReady(readyCtx)
		cancel()
		if e != nil {
			supervisor.Stop()
			log.Fatal(e)
		}
	}
	srv := &http.Server{Handler: a, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		a.Close()
		if supervisor != nil {
			supervisor.Stop()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()
	fmt.Printf("remote-desktop %s listening at %s (data: %s)\n", version, l.Addr(), filepath.Clean(*data))
	if e = srv.Serve(l); e != nil && e != http.ErrServerClosed {
		log.Fatal(e)
	}
}
