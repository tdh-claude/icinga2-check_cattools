package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"os"
)

var (
	arguments docopt.Opts
	err       error

	params struct {
		host     string
		port     int
		username string
		password string
		identity string
		version  bool
		interval int
	}
)

func init() {
	usage := `check_cattools

Check CatTools backup logs

Usage: 
	check_cattools (-h | --help | --version)
	check_cattools [-I <interval> | --interval=<interval>] (-H <host> | --host=<host> -u <username> | --username=<username>) [-p <password> | --password=<password> | -i <pkey_file> | --identity=<pkey_file] [-P <port> | --port=<port>]

Options:
	--version  				Show check_cattools version.
	-h --help  				Show this screen.
	-I <interval> --interval=<interval>  	Interval of backup in day [default: 1]
	-H <host> --host=<host>  		Hostname or IP Address
	-u <username> --username=<username>  	Username
	-p <password> --password=<password>  	Password
	-i <pkey_file> --identity=<pkey_file>  	Private key file [default: ~/.ssh/id_rsa]
	-P <port> --port=<port>  		Port number [default: 22]
`

	arguments, err = docopt.ParseDoc(usage)
	if err != nil {
		fmt.Printf("%s Error parsing command line arguments: %v", UNK, err)
		os.Exit(UNK_CODE)
	}

	params.version, _ = arguments.Bool("--version")
	params.port, _ = arguments.Int("--port")
	params.host, _ = arguments.String("--host")
	params.username, _ = arguments.String("--username")
	params.password, _ = arguments.String("--password")
	params.identity, _ = arguments.String("--identity")
	params.interval, _ = arguments.Int("--interval")
}

func main() {

	if params.version {
		fmt.Println("check_cattools version 0.0.0")
		os.Exit(0)
	}

	ctl := new(CatToolsLog)
	ctl.Load(params.host, params.username, params.password, params.identity, params.port)

	ctl.Analyze(params.interval)
	ctl.ReturnResult()
}
