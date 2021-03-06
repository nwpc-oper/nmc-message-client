package app

import (
	"fmt"
	nmc_message_client "github.com/nwpc-oper/nmc-message-client"
	"github.com/nwpc-oper/nmc-message-client/sender"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"log"
	"os"
	"strings"
	"time"
)

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

var (
	target           = ""
	topic            = ""
	source           = ""
	messageType      = ""
	status           = "0"
	datetime         int64
	fileName         = ""
	absoluteDataName = ""
	debug            = false
	disableSend      = false
	ignoreError      = false
	help             = false
)

func init() {
	rootCmd.AddCommand(sendCmd)
}

var sendCmd = &cobra.Command{
	Use:                "send",
	Short:              "Send message to NMC Monitor",
	Long:               "Send message to NMC Monitor. This command will be deprecated from v0.3.0 and will be dropped out from 2021.",
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		currentTimeStamp := makeTimestamp()

		var sendFlagSet = pflag.NewFlagSet("send", pflag.ContinueOnError)
		sendFlagSet.ParseErrorsWhitelist = pflag.ParseErrorsWhitelist{UnknownFlags: true}
		sendFlagSet.SortFlags = false

		sendFlagSet.StringVar(&target, "target", "", "send targets, split by ','")
		sendFlagSet.StringVar(&topic, "topic", "monitor", "message topic")

		sendFlagSet.StringVar(&source, "source", "", "message source")
		sendFlagSet.StringVar(&messageType, "type", "", "message type")
		sendFlagSet.StringVar(&status, "status", "0", "status")
		sendFlagSet.Int64Var(&datetime, "datetime", currentTimeStamp, "datetime, default is current time.")
		sendFlagSet.StringVar(&fileName, "file-name", "", "file name")
		sendFlagSet.StringVar(&absoluteDataName, "absolute-data-name", "", "absolute data name")

		sendFlagSet.BoolVar(&debug, "debug", false, "show debug information")
		sendFlagSet.BoolVar(&disableSend, "disable-send", false, "disable message send.")
		sendFlagSet.BoolVar(&ignoreError, "ignore-error", false,
			"ignore error. Should be open in operation systems.")

		sendFlagSet.BoolVar(&help, "help", false,
			"show help information.")

		if err := sendFlagSet.Parse(args); err != nil {
			cmd.Usage()
			log.Fatal(err)
		}

		// check if there are non-flag arguments in the command line
		cmds := sendFlagSet.Args()
		if len(cmds) > 0 {
			cmd.Usage()
			log.Fatalf("unknown command: %s", cmds[0])
		}

		// short-circuit on help
		help, err := sendFlagSet.GetBool("help")
		if err != nil {
			log.Fatal(`"help" flag is non-bool, programmer error, please correct`)
		}

		if help {
			cmd.Help()
			fmt.Printf("%s\n", sendFlagSet.FlagUsages())
			return
		}

		// check required flags
		if target == "" {
			log.Fatal("target option is required")
		}
		targetBrokers := strings.Split(target, ",")
		fmt.Printf("brokers: %s\n", targetBrokers)

		if source == "" {
			log.Fatal("source option is required")
		}
		if messageType == "" {
			log.Fatal("messageType option is required")
		}

		if debug {
			fmt.Printf("Version %s (%s)\n", Version, GitCommit)
			fmt.Printf("Build at %s\n", BuildTime)
		}

		var monitorMessageBlob []byte

		if messageType == "prod_grib" {
			monitorMessageBlob, err = generateProdGribMessage(args)
		} else {
			log.Fatalf("message type is not supported: %s", messageType)
		}

		if err != nil {
			f := os.Stderr
			returnCode := 2
			if ignoreError {
				f = os.Stdout
				returnCode = 0
			}
			fmt.Fprintf(f, "create message error: %s\n", err)
			os.Exit(returnCode)
		}

		if debug {
			fmt.Printf("message:\n")
			fmt.Printf("%s\n", monitorMessageBlob)
		}

		if disableSend {
			if debug {
				fmt.Printf("disable send.\n")
				fmt.Printf("Bye.\n")
			}
			return
		}

		var s sender.Sender
		s = &sender.KafkaSender{
			Target: sender.KafkaTarget{
				Brokers:      targetBrokers,
				Topic:        topic,
				WriteTimeout: 10 * time.Second,
			},
			Debug: debug,
		}

		err = s.SendMessage(monitorMessageBlob)

		if err != nil {
			f := os.Stderr
			returnCode := 4
			if ignoreError {
				f = os.Stdout
				returnCode = 0
			}
			fmt.Fprintf(f, "send message failed: %s\n", err)
			os.Exit(returnCode)
		}
	},
}

func generateProdGribMessage(args []string) ([]byte, error) {
	var startTime = ""
	var forecastTime = ""

	var prodGribFlagSet = pflag.NewFlagSet("prod_grid", pflag.ContinueOnError)
	prodGribFlagSet.ParseErrorsWhitelist = pflag.ParseErrorsWhitelist{UnknownFlags: true}
	prodGribFlagSet.SortFlags = false

	prodGribFlagSet.StringVar(&startTime, "start-time", "", "start time, such as 2019062400")
	prodGribFlagSet.StringVar(&forecastTime, "forecast-time", "", "forecast time, such as 000")
	if err := prodGribFlagSet.Parse(args); err != nil {
		log.Fatalf("argument parse fail: %s", err)
	}

	monitorMessageBlob, err := nmc_message_client.CreateProdGribMessage(
		source,
		messageType,
		status,
		datetime,
		fileName,
		absoluteDataName,
		startTime,
		forecastTime)

	return monitorMessageBlob, err
}
