// Command x360-profile serves the Xbox 360 profile REST API (see internal/profile).
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/openpak/x360-common/accountcore"
	"github.com/openpak/x360-common/xstoken"
	"github.com/openpak/x360-profile/internal/profile"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	addr := os.Getenv("X360_PROFILE_ADDR")
	if addr == "" {
		addr = ":10042"
	}
	keyFile := os.Getenv("X360_PROFILE_XSTS_KEY") // the token service's signing key, read-only
	key, err := os.ReadFile(keyFile)
	if err != nil || len(key) < 16 {
		log.Error("xsts signing key (X360_PROFILE_XSTS_KEY)", "err", err, "len", len(key))
		os.Exit(2)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("listen", "err", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Info("x360-profile listening", "addr", ln.Addr())
	handler := &profile.Handler{Signer: &xstoken.Signer{Key: key}, Log: log}
	if coreURL := os.Getenv("X360_PROFILE_ACCOUNT_CORE_URL"); coreURL != "" {
		core, err := accountcore.New(coreURL, os.Getenv("X360_PROFILE_ACCOUNT_CORE_TOKEN"), nil)
		if err != nil {
			log.Error("account core", "err", err)
			os.Exit(2)
		}
		state := os.Getenv("X360_PROFILE_STATE")
		if state == "" {
			log.Error("X360_PROFILE_STATE is required with account media")
			os.Exit(2)
		}
		handler.Core = core
		handler.Pictures = &profile.PictureStore{Dir: state, Key: key}
	}
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	// XAM sends its bearer token only to HTTPS URLs. Keep the HTTP listener
	// for diagnostics and serve console traffic through TLS passthrough.
	var tlsSrv *http.Server
	if tlsAddr := os.Getenv("X360_SERVICE_TLS_ADDR"); tlsAddr != "" {
		cert, err := tls.LoadX509KeyPair(os.Getenv("X360_SERVICE_TLS_CERT"), os.Getenv("X360_SERVICE_TLS_KEY"))
		if err != nil {
			log.Error("tls certificate", "err", err)
			os.Exit(2)
		}
		tlsLn, err := net.Listen("tcp", tlsAddr)
		if err != nil {
			log.Error("tls listen", "err", err)
			os.Exit(1)
		}
		tlsSrv = &http.Server{Handler: srv.Handler, ReadHeaderTimeout: 10 * time.Second,
			TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12, MaxVersion: tls.VersionTLS12,
				Certificates: []tls.Certificate{cert},
				// XAM offers RSA key exchange with AES-CBC, without ECDHE.
				CipherSuites: []uint16{tls.TLS_RSA_WITH_AES_128_CBC_SHA256, tls.TLS_RSA_WITH_AES_128_CBC_SHA}},
			TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){}}
		log.Info("tls listening", "addr", tlsLn.Addr())
		go func() {
			if err := tlsSrv.ServeTLS(tlsLn, "", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Error("tls serve", "err", err)
				os.Exit(1)
			}
		}()
	}
	go func() {
		<-ctx.Done()
		sc, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if tlsSrv != nil {
			tlsSrv.Shutdown(sc)
		}
		srv.Shutdown(sc)
	}()
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("serve", "err", err)
		os.Exit(1)
	}
}
