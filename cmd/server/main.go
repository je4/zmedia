package main

import (
	"context"
	"database/sql"
	"flag"
	"github.com/je4/zmedia/v2/pkg/database"
	"github.com/je4/zmedia/v2/pkg/filesystem"
	"github.com/je4/zmedia/v2/pkg/mediaserver"
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
			log.Panicf("cannot open file %s: %v", config.AccessLog, err)
			return
		}
		defer f.Close()
		accesslog = f
	}

	var fss map[string]filesystem.FileSystem = make(map[string]filesystem.FileSystem)

	for _, s3 := range config.S3 {
		fs, err := filesystem.NewS3Fs(s3.Endpoint, s3.AccessKeyId, s3.SecretAccessKey, s3.UseSSL)
		if err != nil {
			log.Fatalf("cannot connect to s3 instance %v: %v", s3.Name, err)
			return
		}
		fss[s3.Name] = fs
	}

	// get database connection handle
	db, err := sql.Open(config.DB.ServerType, config.DB.DSN)
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}
	defer db.Close()

	// Open doesn't open a connection. Validate DSN data:
	err = db.Ping()
	if err != nil {
		log.Fatalf("error pinging database: %v", err)
	}

	pgdb, err := database.NewPostgresDB(db, "public", log)
	if err != nil {
		log.Panicf("cannot create postgres database layer: %v", err)
		return
	}

	mediadb, err := database.NewMediaDatabase(pgdb, fss)
	if err != nil {
		log.Panicf("cannot create mediaserver database layer: %v", err)
		return
	}

	_, err = mediadb.GetEstateByName("mediathek")
	//_, err = mediadb.CreateEstate("mediathek", "Mediathek HGK")
	if err != nil {
		log.Panicf("cannot create storage: %v", err)
		return
	}

	//_, err = mediadb.CreateStorage("test", "hgk", "")
	_, err = mediadb.GetStorageByName("test")
	if err != nil {
		log.Panicf("cannot get storage: %v", err)
		return
	}

	//_, err = mediadb.CreateCollection("test", est, stor, "test-", "testing 123", 0)
	coll, err := mediadb.GetCollectionByName("test")
	if err != nil {
		log.Panicf("cannot create collection: %v", err)
		return
	}
	coll, err = mediadb.GetCollectionByName("test")
	if err != nil {
		log.Panicf("cannot create collection: %v", err)
		return
	}
	log.Infof("Collection #%d - %s", coll.Id, coll.Name)

	return

	srv, err := mediaserver.NewServer(
		config.HTTPSAddr, config.HTTP3Addr,
		config.HTTPSAddrExt, config.HTTP3AddrExt,
		config.DataPrefix,
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

	go func() {
		if err := srv.ListenAndServe(config.CertPEM, config.KeyPEM); err != nil {
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
