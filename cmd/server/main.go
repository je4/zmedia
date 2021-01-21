package main

import (
	"context"
	"database/sql"
	"flag"
	"github.com/je4/zmedia/v2/pkg/database"
	"github.com/je4/zmedia/v2/pkg/filesystem"
	"github.com/je4/zmedia/v2/pkg/mediaserver"
	sshtunnel "github.com/je4/zmedia/v2/pkg/sshTunnel"
	_ "github.com/lib/pq"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfgfile := flag.String("cfg", "./search.toml", "locations of config file")
	flag.Parse()
	config := LoadConfig(*cfgfile)

	// create logger instance
	log, lf := mediaserver.CreateLogger("memostream", config.Logfile, config.Loglevel)
	defer lf.Close()

	var accesslog io.Writer
	if config.AccessLog == "" {
		accesslog = os.Stdout
	} else {
		f, err := os.OpenFile(config.AccessLog, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			log.Errorf("cannot open file %s: %v", config.AccessLog, err)
			return
		}
		defer f.Close()
		accesslog = f
	}

	if config.SSHTunnel.User != "" && config.SSHTunnel.PrivateKey != "" {
		tunnels := map[string]*sshtunnel.SourceDestination{}
		tunnels["postgres"] = &sshtunnel.SourceDestination{
			Local: &sshtunnel.Endpoint{
				Host: config.SSHTunnel.LocalEndpoint.Host,
				Port: config.SSHTunnel.LocalEndpoint.Port,
			},
			Remote: &sshtunnel.Endpoint{
				Host: config.SSHTunnel.RemoteEndpoint.Host,
				Port: config.SSHTunnel.RemoteEndpoint.Port,
			},
		}
		tunnel, err := sshtunnel.NewSSHTunnel(
			config.SSHTunnel.User,
			config.SSHTunnel.PrivateKey,
			&sshtunnel.Endpoint{
				Host: config.SSHTunnel.ServerEndpoint.Host,
				Port: config.SSHTunnel.ServerEndpoint.Port,
			},
			tunnels,
			log,
		)
		if err != nil {
			log.Errorf("cannot create sshtunnel %v@%v:%v - %v", config.SSHTunnel.User, config.SSHTunnel.ServerEndpoint.Host, &config.SSHTunnel.ServerEndpoint.Port, err)
			return
		}
		if err := tunnel.Start(); err != nil {
			log.Errorf("cannot create sshtunnel %v - %v", tunnel.String(), err)
			return
		}
		defer tunnel.Close()
		time.Sleep(2 * time.Second)
	}

	var fss []filesystem.FileSystem
	for _, s3 := range config.S3 {
		fs, err := filesystem.NewS3Fs(s3.Name, s3.Endpoint, s3.AccessKeyId, s3.SecretAccessKey, s3.UseSSL)
		if err != nil {
			log.Errorf("cannot connect to s3 instance %v: %v", s3.Name, err)
			return
		}
		fss = append(fss, fs)
	}
	for _, f := range config.FileMap {
		fs, err := filesystem.NewLocalFs(f.Alias, f.Folder, log)
		if err != nil {
			log.Errorf("cannot load local filesystem instance %v: %v", f.Alias, err)
			return
		}
		fss = append(fss, fs)
	}

	// get database connection handle
	db, err := sql.Open(config.DB.ServerType, config.DB.DSN)
	if err != nil {
		log.Errorf("error opening database: %v", err)
		return
	}
	defer db.Close()

	// Open doesn't open a connection. Validate DSN data:
	err = db.Ping()
	if err != nil {
		log.Errorf("error pinging database: %v", err)
		return
	}

	pg, err := database.NewPostgresDB(db, config.DB.Schema, log)
	if err != nil {
		log.Errorf("error creating PostgresDB: %v", err)
		return
	}

	idx, err := mediaserver.NewIndexer(
		nil,
		config.Indexer.Siegfried,
		config.Indexer.FFProbe,
		config.Indexer.Identify,
		config.Indexer.Convert,
		config.Indexer.IdentTimeout.Duration,
		"",
	)
	if err != nil {
		log.Errorf("cannot instantiate indexer: %v", err)
		return
	}

	mdb, err := database.NewMediaDatabase(pg, fss...)
	if err != nil {
		log.Errorf("cannot instantiate mediadatabase: %v", err)
		return
	}

	srv, err := mediaserver.NewServerHTTP3(
		config.HTTPSAddr,
		config.HTTPSAddrExt,
		config.DataPrefix,
		config.MediaPrefix,
		config.StaticPrefix,
		config.StaticFolder,
		config.JWTKey,
		config.JWTAlg,
		config.LinkTokenExp.Duration,
		log,
		accesslog)

	if err != nil {
		log.Panicf("cannot create server: %v", err)
		return
	}

	/*
		stor, err := mdb.CreateStorage("test", "s3://hgk", "")
		if err != nil {
			log.Panicf("cannot create storage: %v", err)
			return
		}

		est, err := mdb.CreateEstate("test", "lorem ipsum dolor sit amet")
		if err != nil {
			log.Panicf("cannot create estate: %v", err)
			return
		}

		coll, err := mdb.CreateCollection("test", est, stor, "test-", "testing 123", 0)
		if err != nil {
			log.Panicf("cannot create collection: %v", err)
			return
		}
	*/
	/*
		coll, err := mdb.GetCollectionByName("test")
		if err != nil {
			log.Panicf("cannot load collection: %v", err)
			return
		}

		master, err := mdb.CreateMaster(coll, "testing", "file://test/test.png", nil)
		if err != nil {
			log.Panicf("cannot create master: %v", err)
			return
		}
		log.Infof("%v", master)
	*/

	mh, err := mediaserver.NewMediaHandler(config.MediaPrefix, mdb, idx, log, fss...)
	if err != nil {
		log.Errorf("cannot create media handler: %v", mh)
		return
	}

	go func() {
		if err := srv.ListenAndServeHTTP3(config.CertPEM, config.KeyPEM, mh); err != nil {
			log.Errorf("services ended: %v", err)
		}
	}()
	end := make(chan bool, 1)

	// process waiting for interrupt signal (TERM or KILL)
	go func() {
		sigint := make(chan os.Signal, 1)

		// interrupt signal sent from terminal
		signal.Notify(sigint, os.Interrupt)

		signal.Notify(sigint, syscall.SIGTERM)
		signal.Notify(sigint, syscall.SIGKILL)

		<-sigint

		// We received an interrupt signal, shut down.
		log.Infof("shutdown requested")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		srv.Shutdown(ctx)

		end <- true
	}()

	<-end
	log.Info("server stopped")

}
