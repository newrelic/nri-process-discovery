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
	"time"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/infrastructure-agent/pkg/integrations/cmdrequest/protocol"
	"github.com/newrelic/nri-process-discovery/pkg/lang"
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
		// flush payloads
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

	runID := newRunID()
	processesDiscovered := 0
	for langID, processes := range langs {
		// a cmd request batch could be created, but this streaming approach gets faster time to
		// glass and scatters load when spawning new integrations
		for _, p := range processes {
			payload, err := newSingleCmdReqPayload(langID.IntegrationName(), p.Pid, runID)
			if err != nil {
				l.Errorf("cannot create cmd request for PID %d (lang: %s), err: %s", p.Pid, langID, err.Error())
				continue
			}
			fmt.Println(payload)
			processesDiscovered++
		}
	}

	if err = event(i, processesDiscovered, runID); err != nil {
		l.Errorf(err.Error())
	}
}

func event(i *integration.Integration, processesDiscovered int, runID string) error {
	e, err := i.NewEvent(time.Now(), "dynamic_instrumentation", "start_process_discovery")
	if err != nil {
		return fmt.Errorf("cannot create event, error: %s", err.Error())
	}

	err = e.AddAttribute("processes.discovered", processesDiscovered)
	if err != nil {
		return fmt.Errorf("cannot add attribute to event, error: %s", err.Error())
	}
	err = e.AddAttribute("run.id", runID)
	if err != nil {
		return fmt.Errorf("cannot add attribute to event, error: %s", err.Error())
	}

	i.HostEntity.AddEvent(e)
	return nil
}

func newSingleCmdReqPayload(integrationName string, pid int32, runID string) (payload string, err error) {
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
					"-runId",
					runID,
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

// generates unique (enough) identifier for the process discovery run triggering the LSI integrations.
func newRunID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}
