package main

import (
	"hypefast-api/bootstrap"
	"hypefast-api/lib/utils"
	"hypefast-api/services/api"

	"fmt"
	"log"
	"os"

	"path/filepath"
	"runtime"

	"github.com/urfave/cli/v2"
)

var (
	_, b, _, _ = runtime.Caller(0)
	basepath   = filepath.Dir(b)
	config     utils.Config
	debug      = false

	// app the base of skeleton
	app *bootstrap.App
)

// EnvConfigPath environtment variable that set the config path
const EnvConfigPath = "REBEL_CLI_CONFIG_PATH"

// setup initialize the used variable and dependencies
func setup() {
	configFile := os.Getenv(EnvConfigPath)
	if configFile == "" {
		configFile = "./config.json"
	}

	log.Println(configFile)

	config = utils.NewViperConfig(basepath, configFile)

	debug = config.GetBool("app.debug")
	validator := bootstrap.SetupValidator(config)
	cLog := bootstrap.SetupLogger(config)

	// connect to default redis
	rd, err := bootstrap.SetupRedis(
		config.GetString("db.redis.addr"),
		config.GetString("db.redis.password"),
		config.GetInt("db.redis.default_db"),
	)
	if err != nil {
		fmt.Println(err.Error())
	}

	// connect to redis cache
	rdCache, err := bootstrap.SetupRedis(
		config.GetString("db.redis.addr"),
		config.GetString("db.redis.password"),
		1,
	)
	if err != nil {
		fmt.Println("[redis-cache] " + err.Error())
	}

	app = &bootstrap.App{
		Debug:      debug,
		Config:     config,
		Validator:  validator,
		Log:        cLog,
		Redis:      rd,
		RedisCache: rdCache,
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	setup()
	cmd := &cli.App{
		Name:  "Hypefast Core",
		Usage: "Hypefast Core, cli",
		Commands: []*cli.Command{
			{
				Name:   "api",
				Usage:  "API service. Run on http 1/1",
				Flags:  api.Flags,
				Action: api.Boot{App: app}.Start,
			},
		},
		Action: func(cli *cli.Context) error {
			fmt.Printf("%s version:%s\n", cli.App.Name, "1.0")
			return nil
		},
	}

	err := cmd.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
