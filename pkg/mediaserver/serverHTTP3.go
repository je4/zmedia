package mediaserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/goph/emperror"
	"github.com/gorilla/mux"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/op/go-logging"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

type ServerHTTP3 struct {
	_srv         *http.Server
	srv3         *http3.Server
	host         string
	port         string
	httpAddrExt  string
	dataPrefix   string
	jwtKey       string
	jwtAlg       []string
	linkTokenExp time.Duration
	log          *logging.Logger
	accesslog    io.Writer
	mediaPrefix  string
	staticPrefix string
	staticFolder string
}

func NewServerHTTP3(
	addr string,
	httpAddrExt string,
	dataPrefix string,
	mediaPrefix string,
	staticPrefix string,
	staticFolder string,
	jwtKey string,
	jwtAlg []string,
	linkTokenExp time.Duration,
	log *logging.Logger,
	accesslog io.Writer) (*ServerHTTP3, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		//log.Panicf("cannot split address %s: %v", addr, err)
		return nil, emperror.Wrapf(err, "cannot split address %s", addr)
	}
	/*
		host3, port3, err := net.SplitHostPort(addr3)
		if err != nil {
			//log.Panicf("cannot split address %s: %v", addr, err)
			return nil, emperror.Wrapf(err, "cannot split address %s", addr)
		}
	*/
	srv := &ServerHTTP3{
		host:         host,
		port:         port,
		httpAddrExt:  httpAddrExt,
		dataPrefix:   dataPrefix,
		mediaPrefix:  mediaPrefix,
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

func ListenAndServeHTTP3(addr string, certs []tls.Certificate, handler http.Handler) error {
	// Load certs
	var err error
	/*
		certs := make([]tls.Certificate, 1)
		certs[0], err = tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return err
		}
	*/

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

func (s *ServerHTTP3) ListenAndServeHTTP3(cert, key string, mh *MediaHandler) error {
	staticPrefix := fmt.Sprintf("/%v/", s.staticPrefix)

	routerHTTP3 := mux.NewRouter()
	routerHTTP3.PathPrefix(staticPrefix).
		Handler(http.StripPrefix(staticPrefix, http.FileServer(http.Dir(s.staticFolder))))
		//	loggedRouter3 := handlers.CombinedLoggingHandler(s.accesslog, routerHTTP3)

	/*
		sub := routerHTTP3.PathPrefix(fmt.Sprintf("/%s/", s.mediaPrefix)).Subrouter()
	*/
	if err := mh.SetRoutes(routerHTTP3); err != nil {
		return emperror.Wrap(err, "cannot initialize subroutes")
	}

	var certs []tls.Certificate
	if cert == "auto" || key == "auto" {
		s.log.Info("generating new certificate")
		cert, err := DefaultCertificate()
		if err != nil {
			return emperror.Wrap(err, "cannot generate default certificate")
		}
		certs = []tls.Certificate{*cert}
	} else if cert != "" && key != "" {
		cert, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			return emperror.Wrapf(err, "cannot load key pair %v/%v", cert, key)
		}
		certs = []tls.Certificate{cert}
	}

	var addr = fmt.Sprintf("%s:%s", s.host, s.port)
	var wg sync.WaitGroup
	var http3Err error
	wg.Add(1)
	go func() {
		s.log.Infof("starting HTTP3 zsearch at https://%s/%s", addr, s.mediaPrefix)
		//		http3Err = listenAndServeHTTP3(addr, certs, loggedRouter3) // s.srv3.ListenAndServe()
		err := ListenAndServeHTTP3(addr, certs, routerHTTP3)
		if err != nil {
			s.log.Errorf("server not started: %v", err)
		}

		s.log.Infof("HTTP3 service stopped: %v", http3Err)
		wg.Done()
	}()
	wg.Wait()

	return http3Err
}

func (s *ServerHTTP3) Shutdown(ctx context.Context) {
	if s.srv3 != nil {
		s.srv3.Shutdown(ctx)
	}
}
