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
	DataPrefix         string       `toml:"mediaprefix"`
	StaticPrefix       string       `toml:"staticprefix"`
	StaticFolder       string       `toml:"staticfolder"`
	DBOld              Cfg_database `toml:"dbold"`
	DB                 Cfg_database `toml:"dbold"`
	S3                 []Cfg_S3     `toml:"s3"`
}

func LoadConfig(fp string) Config {
	var conf Config
	_, err := toml.DecodeFile(fp, &conf)
	if err != nil {
		log.Fatalln("Error on loading config: ", err)
	}
	//fmt.Sprintf("%v", m)
	conf.DataPrefix = strings.Trim(conf.DataPrefix, "/")
	conf.StaticPrefix = strings.Trim(conf.StaticPrefix, "/")
	conf.HTTPSAddrExt = strings.TrimRight(conf.HTTPSAddrExt, "/")
	conf.HTTP3AddrExt = strings.TrimRight(conf.HTTP3AddrExt, "/")
	conf.StaticFolder = filepath.Clean(conf.StaticFolder)
	return conf
}
