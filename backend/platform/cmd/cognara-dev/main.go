// cognara-dev is an explicit loopback-only development host. It never starts Python DocReader or a sandbox.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	agentruntime "github.com/Tencent/WeKnora/internal/platform/adapters/runtime"
	api "github.com/Tencent/WeKnora/internal/platform/api"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/connectors/cognitive"
	"github.com/Tencent/WeKnora/internal/platform/mockae"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	port := flag.Int("port", 8080, "loopback HTTP port")
	mockPort := flag.Int("mock-port", 8081, "A-E contract fixture port")
	storage := flag.String("storage", "json", "json or sqlite local persistence")
	data := flag.String("data", ".devdata/state.json", "local F/G persistence path")
	modelConfig := flag.String("model-config", "", "non-secret model registry JSON; credentials are environment references")
	cognitiveURL := flag.String("cognitive-url", "", "external A-E base URL; empty starts the local fixture")
	demo := flag.Bool("demo", false, "explicitly allow local validation identities")
	flag.Parse()
	if !*demo {
		log.Fatal("This entry is development-only. Pass --demo; use the upstream host for production integration.")
	}
	var store *app.Store
	var e error
	switch *storage {
	case "json":
		store, e = app.Open(*data)
	case "sqlite":
		store, e = app.OpenSQLite(*data)
	default:
		log.Fatal("unsupported storage adapter")
	}
	if e != nil {
		log.Fatal(e)
	}
	defer store.Close()
	key := make([]byte, 32)
	if _, e = rand.Read(key); e != nil {
		log.Fatal(e)
	}
	token := hex.EncodeToString(key)
	if *cognitiveURL != "" {
		token = os.Getenv("COGNARA_SERVICE_TOKEN")
		if token == "" {
			log.Fatal("external A-E requires COGNARA_SERVICE_TOKEN")
		}
	}
	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	mockAddr := fmt.Sprintf("127.0.0.1:%d", *mockPort)
	baseURL := "http://" + mockAddr
	if *cognitiveURL != "" {
		baseURL = *cognitiveURL
	}
	client := &cognitive.Client{BaseURL: baseURL, ServiceToken: token, SigningKey: key}
	models, err := agentruntime.Load(*modelConfig)
	if err != nil {
		log.Fatal(err)
	}
	server := &api.Server{Models: models, Store: store, Client: client, ServiceToken: token, SigningKey: key}
	servers := []*http.Server{{Addr: addr, Handler: server.Handler(), ReadHeaderTimeout: 5 * time.Second}}
	if *cognitiveURL == "" {
		servers = append(servers, &http.Server{Addr: mockAddr, Handler: mockae.New("http://"+addr, token).Handler(), ReadHeaderTimeout: 5 * time.Second})
	}
	listeners := []net.Listener{}
	for _, s := range servers {
		l, err := net.Listen("tcp", s.Addr)
		if err != nil {
			for _, old := range listeners {
				old.Close()
			}
			log.Fatal(err)
		}
		listeners = append(listeners, l)
	}
	for i, s := range servers {
		go func(s *http.Server, l net.Listener) {
			if e := s.Serve(l); e != nil && e != http.ErrServerClosed {
				log.Print(e)
			}
		}(s, listeners[i])
	}
	log.Printf("Cognara validation API http://%s (validation identities; model execution follows the application release)", addr)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, s := range servers {
		_ = s.Shutdown(ctx)
	}
}
