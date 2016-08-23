package config

import (
	"github.com/BurntSushi/toml"
	"github.com/allen13/golerta/app/auth/ldap"
	"github.com/allen13/golerta/app/db/rethinkdb"
	"github.com/allen13/golerta/app/notifiers"
	"log"
	"time"
)

type GolertaConfig struct {
	Golerta   golerta
	Ldap      ldap.LDAPAuthProvider
	Rethinkdb rethinkdb.RethinkDB
	Notifiers notifiers.Notifiers
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

type golerta struct {
	SigningKey              string   `toml:"signing_key"`
	AuthProvider            string   `toml:"auth_provider"`
	ContinuousQueryInterval duration `toml:"continuous_query_interval"`
}

func BuildConfig(configFile string) (config GolertaConfig) {
	_, err := toml.DecodeFile(configFile, &config)

	if err != nil {
		log.Fatal("config file error: " + err.Error())
	}

	setDefaultConfigs(&config)
	return
}

func setDefaultConfigs(config *GolertaConfig) {
	if config.Golerta.AuthProvider == "" {
		config.Golerta.AuthProvider = "ldap"
	}
	if config.Golerta.ContinuousQueryInterval.Duration == 0 {
		config.Golerta.ContinuousQueryInterval.Duration = time.Second * 5
	}
}
