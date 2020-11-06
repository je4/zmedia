package mediaserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/goph/emperror"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/op/go-logging"
	"go.uber.org/multierr"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	srv          *http.Server
	srv3         *http3.Server
	host         string
	port         string
	host3        string
	port3        string
	httpAddrExt  string
	http3AddrExt string
	dataPrefix   string
	jwtKey       string
	jwtAlg       []string
	linkTokenExp time.Duration
	log          *logging.Logger
	accesslog    io.Writer
	staticPrefix string
	staticFolder string
}

func NewServer(
	addr string,
	addr3 string,
	httpAddrExt string,
	http3AddrExt string,
	dataPrefix string,
	staticPrefix string,
	staticFolder string,
	jwtKey string,
	jwtAlg []string,
	linkTokenExp time.Duration,
	log *logging.Logger,
	accesslog io.Writer) (*Server, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		//log.Panicf("cannot split address %s: %v", addr, err)
		return nil, emperror.Wrapf(err, "cannot split address %s", addr)
	}
	host3, port3, err := net.SplitHostPort(addr3)
	if err != nil {
		//log.Panicf("cannot split address %s: %v", addr, err)
		return nil, emperror.Wrapf(err, "cannot split address %s", addr)
	}
	srv := &Server{
		host:         host,
		port:         port,
		host3:        host3,
		port3:        port3,
		httpAddrExt:  httpAddrExt,
		http3AddrExt: http3AddrExt,
		dataPrefix:   dataPrefix,
		staticPrefix: staticPrefix,
		staticFolder: staticFolder,
		jwtKey:       jwtKey,
		jwtAlg:       jwtAlg,
		linkTokenExp: linkTokenExp,
		log:          log,
		accesslog:    accesslog,
	}
	return srv, nil
}

// ListenAndServe listens on the given network address for both, TLS and QUIC
// connetions in parallel. It returns if one of the two returns an error.
// http.DefaultServeMux is used when handler is nil.
// The correct Alt-Svc headers for QUIC are set.
func ListenAndServe(addr, certFile, keyFile string, handler http.Handler) error {
	// Load certs
	var err error
	certs := make([]tls.Certificate, 1)
	certs[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}
	return listenAndServe(addr, certs, handler)
}

func listenAndServe(addr string, certs []tls.Certificate, handler http.Handler) error {
	// We currently only use the cert-related stuff from tls.Config,
	// so we don't need to make a full copy.
	config := &tls.Config{
		Certificates: certs,
	}

	// Open the listeners
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	defer udpConn.Close()

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}
	tcpConn, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}
	defer tcpConn.Close()

	tlsConn := tls.NewListener(tcpConn, config)
	defer tlsConn.Close()

	// Start the servers
	httpServer := &http.Server{
		Addr:      addr,
		TLSConfig: config,
	}

	quicServer := &http3.Server{
		Server: httpServer,
	}

	if handler == nil {
		handler = http.DefaultServeMux
	}
	httpServer.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		quicServer.SetQuicHeaders(w.Header())
		handler.ServeHTTP(w, r)
	})

	hErr := make(chan error)
	qErr := make(chan error)
	go func() {
		hErr <- httpServer.Serve(tlsConn)
	}()
	go func() {
		qErr <- quicServer.Serve(udpConn)
	}()

	select {
	case err := <-hErr:
		quicServer.Close()
		return err
	case err := <-qErr:
		// Cannot close the HTTP server or wait for requests to complete properly :/
		return err
	}
}

func (s *Server) ListenAndServe(cert, key string) error {
	routerHTTP := mux.NewRouter()
	staticPrefix := fmt.Sprintf("/%v/", s.staticPrefix)
	routerHTTP.PathPrefix(staticPrefix).
		Handler(http.StripPrefix(staticPrefix, http.FileServer(http.Dir(s.staticFolder))))
	routerHTTP.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Alt-Svc", fmt.Sprintf(`h3=":%v"; ma=86400,h3-23=":%v"; ma=86400,h3-27=":%v"; ma=86400, h3-28=":%v"; ma=86400, h3-29=":%v"; ma=86400, quic=":%v"; ma=86400`, s.port3, s.port3, s.port3, s.port3, s.port3, s.port3))
			// Pass down the request to the next middleware (or final handler)
			next.ServeHTTP(w, r)
		})
	})

	loggedRouter := handlers.CombinedLoggingHandler(s.accesslog, routerHTTP)
	addr := net.JoinHostPort(s.host, s.port)
	s.srv = &http.Server{
		Handler: loggedRouter,
		Addr:    addr,
	}

	routerHTTP3 := mux.NewRouter()
	routerHTTP3.PathPrefix(staticPrefix).
		Handler(http.StripPrefix(staticPrefix, http.FileServer(http.Dir(s.staticFolder))))
	loggedRouter3 := handlers.CombinedLoggingHandler(s.accesslog, routerHTTP3)
	addr3 := net.JoinHostPort(s.host3, s.port3)

	var certs []tls.Certificate

	if cert == "" && key == "" {
		s.log.Infof("starting HTTP zsearch at http://%v", addr)
		return s.srv.ListenAndServe()
	} else {
		if cert == "auto" || key == "auto" {
			s.log.Info("generating new certificate")
			cert, err := DefaultCertificate()
			if err != nil {
				return emperror.Wrap(err, "cannot generate default certificate")
			}
			certs = []tls.Certificate{*cert}
			//			s.srv.TLSConfig = &tls.Config{Certificates: certs}
			//			s.srv3.TLSConfig = &tls.Config{Certificates: certs}

		} else if cert != "" && key != "" {
			cert, err := tls.LoadX509KeyPair(cert, key)
			if err != nil {
				return emperror.Wrapf(err, "cannot load key pair %v/%v", cert, key)
			}
			certs = []tls.Certificate{cert}
			//			s.srv.TLSConfig = &tls.Config{Certificates: certs}
			//			s.srv3.TLSConfig = &tls.Config{Certificates: certs}
		}
	}

	var wg sync.WaitGroup
	var httpErr, http3Err error
	wg.Add(1)
	/*
		go func() {
			s.log.Infof("starting HTTPS zsearch at https://%v/%v", addr, s.dataPrefix)
			httpErr = s.srv.ListenAndServeTLS("", "")
			s.log.Infof("HTTPS service stopped: %v", httpErr)
			time.Sleep(time.Second)
			s.srv3.Shutdown(context.Background())
			wg.Done()
		}()
	*/
	go func() {
		s.log.Infof("starting HTTP3 zsearch at %v", addr3)
		http3Err = listenAndServe(addr3, certs, loggedRouter3) // s.srv3.ListenAndServe()

		s.log.Infof("HTTP3 service stopped: %v", http3Err)
		//		time.Sleep(time.Second)
		//		s.srv.Shutdown(context.Background())
		wg.Done()
	}()
	wg.Wait()

	if httpErr == nil {
		return http3Err
	}
	if http3Err == nil {
		return httpErr
	}
	return multierr.Combine(httpErr, http3Err)
}

func (s *Server) Shutdown(ctx context.Context) {
	if s.srv != nil {
		s.srv.Shutdown(ctx)
	}
	if s.srv3 != nil {
		s.srv3.Shutdown(ctx)
	}
}
