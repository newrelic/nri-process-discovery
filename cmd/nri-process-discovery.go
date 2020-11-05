package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/infrastructure-agent/pkg/integrations/cmdrequest/protocol"
	"go.datanerd.us/p/ohai/nri-process-discovery/pkg/lang"
)

const (
	integrationName    = "com.new-relic.nri-process-discovery"
	integrationVersion = "0.0.0"
)

var (
	debug = flag.Bool("debug", false, "Enable debug mode")
)

func main() {
	flag.Parse()

	l := log.NewStdErr(*debug)

	options := []integration.Option{
		integration.Logger(l),
	}

	// infra agent integration
	i, err := integration.New(integrationName, integrationVersion, options...)
	if err != nil {
		// not able to forwards logs to agent
		panic(err)
	}
	defer func() {
		// forward integration logs to agent
		if err = i.Publish(); err != nil {
			panic(err)
		}
	}()

	// runtime cancellation
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGQUIT)
		for {
			<-sigs
			cancel()
		}
	}()

	langs := lang.ProcessesPerLang(ctx, l)

	for langID, processes := range langs {
		// a cmd request batch could be created, but this streaming approach gets faster time to
		// glass and scatters load when spawning new integrations
		for _, p := range processes {
			payload, err := newSingleCmdReqPayload(langID.IntegrationName(), p.Pid)
			if err != nil {
				l.Errorf("cannot create cmd request for PID %d (lang: %s), err: %s", p.Pid, langID, err.Error())
				continue
			}
			fmt.Println(payload)

		}
	}
}

func newSingleCmdReqPayload(integrationName string, pid int32) (payload string, err error) {
	crs := protocol.CmdRequestV1{
		CmdRequestDiscriminator: protocol.CmdRequestDiscriminator{
			CommandRequestVersion: "1",
		},
		Commands: []protocol.CmdRequestV1Cmd{
			{
				Name: integrationName,
				Args: []string{
					"-introspect",
					strconv.Itoa(int(pid)),
				},
			},
		},
	}

	serializedCR, err := json.Marshal(crs)
	if err != nil {
		return
	}

	payload = strings.Replace(string(serializedCR), "\n", "", -1)
	return
}
