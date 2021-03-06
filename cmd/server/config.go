/*
Copyright 2020 info-age GmbH, Basel.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS-IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"github.com/BurntSushi/toml"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

type Cfg_database struct {
	ServerType string
	DSN        string
	ConnMax    int `toml:"connection_max"`
	Schema     string
}

type Cfg_S3 struct {
	Name            string `toml:"name"`
	Endpoint        string `toml:"endpoint"`
	AccessKeyId     string `toml:"accessKeyId"`
	SecretAccessKey string `toml:"secretAccessKey"`
	UseSSL          bool   `toml:"useSSL"`
}

type FileMap struct {
	Alias  string
	Folder string
}

type Endpoint struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

type SSHTunnel struct {
	User           string   `toml:"user"`
	PrivateKey     string   `toml:"privatekey"`
	LocalEndpoint  Endpoint `toml:"localendpoint"`
	ServerEndpoint Endpoint `toml:"serverendpoint"`
	RemoteEndpoint Endpoint `toml:"remoteendpoint"`
}

type Indexer struct {
	Siegfried    string   `toml:"siegfried"`
	FFProbe      string   `toml:"ffprobe"`
	IdentTimeout duration `toml:"identtimeout"`
	Convert      string   `toml:"convert"`
	Identify     string   `toml:"identify"`
}

type Action struct {
	Name   string
	Params []string
}

type Config struct {
	Logfile            string       `toml:"logfile"`
	Loglevel           string       `toml:"loglevel"`
	AccessLog          string       `toml:"accesslog"`
	HTTPSAddr          string       `toml:"httpaddr"`
	HTTP3Addr          string       `toml:"http3addr"`
	HTTPSAddrExt       string       `toml:"httpaddrext"`
	HTTP3AddrExt       string       `toml:"http3addrext"`
	CertPEM            string       `toml:"certpem"`
	KeyPEM             string       `toml:"keypem"`
	StaticCacheControl string       `toml:"staticcachecontrol"`
	JWTKey             string       `toml:"jwtkey"`
	JWTAlg             []string     `toml:"jwtalg"`
	LinkTokenExp       duration     `toml:"linktokenexp"`
	MediaPrefix        string       `toml:"mediaprefix"`
	DataPrefix         string       `toml:"dataprefix"`
	StaticPrefix       string       `toml:"staticprefix"`
	StaticFolder       string       `toml:"staticfolder"`
	FileMap            []FileMap    `toml:"filemap"`
	DBOld              Cfg_database `toml:"dbold"`
	DB                 Cfg_database `toml:"db"`
	S3                 []Cfg_S3     `toml:"s3"`
	SSHTunnel          SSHTunnel    `toml:"sshtunnel"`
	Indexer            Indexer      `toml:"indexer"`
	Tempdir            string       `toml:"tempdir"`
	Tempsize           int64        `toml:"tempsize"`
	Actions            []Action     `toml:"action"`
}

func LoadConfig(fp string) Config {
	var conf Config
	_, err := toml.DecodeFile(fp, &conf)
	if err != nil {
		log.Fatalln("Error on loading config: ", err)
	}
	//fmt.Sprintf("%v", m)
	if conf.Tempdir == "" {
		conf.Tempdir = os.TempDir()
	}
	conf.DataPrefix = strings.Trim(conf.DataPrefix, "/")
	conf.MediaPrefix = strings.Trim(conf.MediaPrefix, "/")
	conf.StaticPrefix = strings.Trim(conf.StaticPrefix, "/")
	conf.HTTPSAddrExt = strings.TrimRight(conf.HTTPSAddrExt, "/")
	conf.HTTP3AddrExt = strings.TrimRight(conf.HTTP3AddrExt, "/")
	conf.StaticFolder = filepath.Clean(conf.StaticFolder)
	return conf
}
