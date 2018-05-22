package main

import (
	nethttp "net/http"
	"os"
	"strings"

	"github.com/influxdata/platform/http"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	influxlogger "github.com/influxdata/influxdb/logger"
)

var transpileCmd = &cobra.Command{
	Use:   "transpilerd",
	Short: "Transpiler Query Server",
	Run:   func(cmd *cobra.Command, args []string) {
		logger := influxlogger.New(os.Stdout)
		if err := transpileF(cmd, logger, args); err != nil {
			logger.Error("encountered fatal error", zap.String("error", err.Error()))
			os.Exit(1)
		}
	},
}

// Flags contains all the CLI flag values for transpilerd.
type Flags struct {
	bindAddr string
}

var flags Flags

func init() {
	viper.SetEnvPrefix("TRANSPILERD")

	transpileCmd.PersistentFlags().StringVar(&flags.bindAddr, "bind-addr", ":8098", "The bind address for this daemon.")
	viper.BindEnv("BIND_ADDR")
	if b := viper.GetString("BIND_ADDR"); b != "" {
		flags.bindAddr = b
	}

	// TODO(jsternberg): Connect directly to the storage hosts. There's no need to require proxying
	// the requests through ifqld for this service.
	transpileCmd.PersistentFlags().String("ifqld-hosts", "http://localhost:8093", "scheme://host:port address of the ifqld server.")
	viper.BindEnv("IFQLD_HOSTS")
	viper.BindPFlag("IFQLD_HOSTS", transpileCmd.PersistentFlags().Lookup("ifqld-hosts"))
}

func transpileF(cmd *cobra.Command, logger *zap.Logger, args []string) error {
	hosts, err := discoverHosts()
	if err != nil {
		return err
	} else if len(hosts) == 0 {
		return errors.New("no ifqld hosts found")
	}

	// TODO(nathanielc): Allow QueryService to use multiple hosts.

	logger.Info("using ifqld service", zap.Strings("hosts", hosts))
	transpileHandler := http.NewTranspilerQueryHandler()
	transpileHandler.QueryService = &http.QueryService{
		Addr: hosts[0],
	}

	//TODO(nathanielc): Add health checks

	handler := http.NewHandler("transpile")
	handler.Handler = transpileHandler

	logger.Info("starting transpilerd", zap.String("bindAddr", flags.bindAddr))
	return nethttp.ListenAndServe(flags.bindAddr, handler)
}

func main() {
	if err := transpileCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func getStrList(key string) ([]string, error) {
	v := viper.GetViper()
	valStr := v.GetString(key)
	if valStr == "" {
		return nil, errors.New("empty value")
	}

	return strings.Split(valStr, ","), nil
}

func discoverHosts() ([]string, error) {
	ifqldHosts, err := getStrList("IFQLD_HOSTS")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ifqld hosts")
	}
	return ifqldHosts, nil
}
